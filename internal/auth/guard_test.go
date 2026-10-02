package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/testredis"
)

// guardHelpers wraps a Guard so the tests read like the rules they check.
func guardHelpers(t *testing.T, g Guard) (check func(string) time.Duration, fail func(string)) {
	check = func(email string) time.Duration {
		d, err := g.Check(t.Context(), email)
		require.NoError(t, err)
		return d
	}
	fail = func(email string) {
		require.NoError(t, g.Fail(t.Context(), email))
	}
	return check, fail
}

func TestLoginGuard(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	g := NewLoginGuard()
	g.now = func() time.Time { return now }
	check, fail := guardHelpers(t, g)

	for range 4 {
		fail("ana@example.com")
	}
	assert.Zero(t, check("ana@example.com"), "4 failures do not lock")

	fail("ANA@example.com ")
	assert.Equal(t, time.Minute, check("ana@example.com"), "5th failure locks for 1 minute, e-mail normalized")
	assert.Zero(t, check("bob@example.com"), "other e-mails are not affected")

	now = now.Add(time.Minute)
	assert.Zero(t, check("ana@example.com"), "lock expires")

	fail("ana@example.com")
	assert.Equal(t, 2*time.Minute, check("ana@example.com"), "each new failure doubles the lock")

	for range 10 {
		fail("ana@example.com")
	}
	assert.Equal(t, 15*time.Minute, check("ana@example.com"), "lock is capped")

	require.NoError(t, g.Success(t.Context(), "ana@example.com"))
	assert.Zero(t, check("ana@example.com"), "success clears the failures")
}

func TestLoginGuard_OldFailuresAreForgotten(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	g := NewLoginGuard()
	g.now = func() time.Time { return now }
	check, fail := guardHelpers(t, g)

	for range 4 {
		fail("ana@example.com")
	}
	now = now.Add(time.Hour)
	fail("ana@example.com")
	assert.Zero(t, check("ana@example.com"))
}

// Redis keeps time itself, so this test uses short locks and checks ranges.
func TestRedisGuard(t *testing.T) {
	client := testredis.New(t)
	g := NewRedisGuard(client)
	g.baseLock = time.Second
	g.maxLock = 4 * time.Second
	check, fail := guardHelpers(t, g)

	for range 4 {
		fail("ana@example.com")
	}
	assert.Zero(t, check("ana@example.com"), "4 failures do not lock")

	fail("ANA@example.com ")
	assert.InDelta(t, time.Second, check("ana@example.com"), float64(200*time.Millisecond), "5th failure locks, e-mail normalized")
	assert.Zero(t, check("bob@example.com"), "other e-mails are not affected")

	fail("ana@example.com")
	assert.InDelta(t, 2*time.Second, check("ana@example.com"), float64(200*time.Millisecond), "each new failure doubles the lock")

	for range 70 {
		fail("ana@example.com")
	}
	assert.InDelta(t, 4*time.Second, check("ana@example.com"), float64(200*time.Millisecond), "lock is capped, even far past the limit")

	keys, err := client.Keys(t.Context(), "*").Result()
	require.NoError(t, err)
	for _, k := range keys {
		assert.NotContains(t, k, "ana", "keys carry a hash, not the e-mail")
	}

	require.NoError(t, g.Success(t.Context(), "ana@example.com"))
	assert.Zero(t, check("ana@example.com"), "success clears the failures")
}
