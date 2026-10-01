package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-with-at-least-32-characters"

func TestTokens_IssueAndParse(t *testing.T) {
	tokens := NewTokens(testSecret, time.Hour)

	raw, err := tokens.Issue(Claims{UserID: 42, Role: RoleDriver})
	require.NoError(t, err)

	claims, err := tokens.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, Claims{UserID: 42, Role: RoleDriver}, claims)
}

func TestTokens_ParseRejects(t *testing.T) {
	tokens := NewTokens(testSecret, time.Hour)
	valid, err := tokens.Issue(Claims{UserID: 1, Role: RoleAdmin})
	require.NoError(t, err)

	expired := NewTokens(testSecret, time.Hour)
	expired.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	expiredToken, err := expired.Issue(Claims{UserID: 1, Role: RoleAdmin})
	require.NoError(t, err)

	otherSecret, err := NewTokens("another-secret-with-at-least-32-chars", time.Hour).Issue(Claims{UserID: 1, Role: RoleAdmin})
	require.NoError(t, err)

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "1", "role": "admin"}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	badRole, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "1", "role": "root"}).
		SignedString([]byte(testSecret))
	require.NoError(t, err)

	cases := map[string]string{
		"expired":      expiredToken,
		"other secret": otherSecret,
		"alg none":     unsigned,
		"unknown role": badRole,
		"garbage":      "not-a-token",
		"tampered":     valid + "x",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tokens.Parse(raw)
			assert.Error(t, err)
		})
	}
}

func TestPassword(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	require.NoError(t, err)
	assert.True(t, CheckPassword(hash, "s3cret-pass"))
	assert.False(t, CheckPassword(hash, "wrong-pass"))
}
