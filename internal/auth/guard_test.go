package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoginGuard(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	g := NewLoginGuard()
	g.now = func() time.Time { return now }

	for range 4 {
		g.Fail("ana@example.com")
	}
	assert.Zero(t, g.Check("ana@example.com"), "4 failures do not lock")

	g.Fail("ANA@example.com ")
	assert.Equal(t, time.Minute, g.Check("ana@example.com"), "5th failure locks for 1 minute, e-mail normalized")
	assert.Zero(t, g.Check("bob@example.com"), "other e-mails are not affected")

	now = now.Add(time.Minute)
	assert.Zero(t, g.Check("ana@example.com"), "lock expires")

	g.Fail("ana@example.com")
	assert.Equal(t, 2*time.Minute, g.Check("ana@example.com"), "each new failure doubles the lock")

	for range 10 {
		g.Fail("ana@example.com")
	}
	assert.Equal(t, 15*time.Minute, g.Check("ana@example.com"), "lock is capped")

	g.Success("ana@example.com")
	assert.Zero(t, g.Check("ana@example.com"), "success clears the failures")
}

func TestLoginGuard_OldFailuresAreForgotten(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	g := NewLoginGuard()
	g.now = func() time.Time { return now }

	for range 4 {
		g.Fail("ana@example.com")
	}
	now = now.Add(time.Hour)
	g.Fail("ana@example.com")
	assert.Zero(t, g.Check("ana@example.com"))
}
