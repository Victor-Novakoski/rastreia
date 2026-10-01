package auth

import (
	"strings"
	"sync"
	"time"
)

// LoginGuard slows down password guessing on a single account. After
// maxFails wrong passwords for the same e-mail, that e-mail is locked for a
// period that doubles with each further failure, up to maxLock. It counts
// e-mails that do not exist too, so a lock does not reveal which accounts
// exist. State lives in memory: enough for one API instance; with several,
// it moves to Redis.
type LoginGuard struct {
	mu       sync.Mutex
	entries  map[string]*attempts
	maxFails int
	baseLock time.Duration
	maxLock  time.Duration
	now      func() time.Time
}

type attempts struct {
	fails       int
	lockedUntil time.Time
	lastFail    time.Time
}

// sweepAt is how many tracked e-mails trigger a cleanup of stale entries.
const sweepAt = 10_000

func NewLoginGuard() *LoginGuard {
	return &LoginGuard{
		entries:  map[string]*attempts{},
		maxFails: 5,
		baseLock: time.Minute,
		maxLock:  15 * time.Minute,
		now:      time.Now,
	}
}

// Check reports how long the e-mail is still locked; zero means it may try.
func (g *LoginGuard) Check(email string) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.entries[key(email)]
	if !ok {
		return 0
	}
	return max(a.lockedUntil.Sub(g.now()), 0)
}

// Fail records a wrong password.
func (g *LoginGuard) Fail(email string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	if len(g.entries) >= sweepAt {
		g.sweep(now)
	}
	a, ok := g.entries[key(email)]
	if !ok {
		a = &attempts{}
		g.entries[key(email)] = a
	}
	// Failures far apart (longer than the max lock) start the count over.
	if now.Sub(a.lastFail) > g.maxLock {
		a.fails = 0
	}
	a.fails++
	a.lastFail = now
	if a.fails >= g.maxFails {
		lock := g.baseLock << (a.fails - g.maxFails)
		if lock > g.maxLock || lock <= 0 {
			lock = g.maxLock
		}
		a.lockedUntil = now.Add(lock)
	}
}

// Success clears the failures of an e-mail after a correct login.
func (g *LoginGuard) Success(email string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.entries, key(email))
}

func (g *LoginGuard) sweep(now time.Time) {
	for k, a := range g.entries {
		if now.After(a.lockedUntil) && now.Sub(a.lastFail) > g.maxLock {
			delete(g.entries, k)
		}
	}
}

func key(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
