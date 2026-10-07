package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/route"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/user"
)

type noUsers struct{}

func (noUsers) GetUserByEmail(context.Context, string) (store.User, error) {
	return store.User{}, pgx.ErrNoRows
}

func newTestServer(opts Options) http.Handler {
	tokens := auth.NewTokens("test-secret-with-at-least-32-characters", time.Hour)
	authHandler := auth.NewHandler(noUsers{}, tokens, auth.NewLoginGuard(), auth.NewSessions(nil, time.Hour), auth.CookieOptions{})
	return New(Deps{
		Tokens:     tokens,
		Auth:       authHandler,
		Users:      user.NewHandler(user.NewService(nil), authHandler),
		Deliveries: delivery.NewHandler(delivery.NewService(nil)),
		Routes:     route.NewHandler(route.NewService(nil, nil)),
		Ready:      func(*http.Request) error { return nil },
		Options:    opts,
	})
}

func get(h http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "203.0.113.7:1234"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSecurityHeaders(t *testing.T) {
	rec := get(newTestServer(Options{}), "/health", nil)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"), "no HSTS outside production")

	rec = get(newTestServer(Options{Production: true}), "/health", nil)
	assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
}

func TestCORS(t *testing.T) {
	h := newTestServer(Options{CORSOrigins: []string{"https://app.rastreia.dev"}})

	rec := get(h, "/health", map[string]string{"Origin": "https://app.rastreia.dev"})
	assert.Equal(t, "https://app.rastreia.dev", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"), "the refresh cookie needs it")

	rec = get(h, "/health", map[string]string{"Origin": "https://evil.example"})
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestRateLimit(t *testing.T) {
	h := newTestServer(Options{RateLimit: 3})
	for range 3 {
		require.Equal(t, http.StatusOK, get(h, "/health", nil).Code)
	}
	rec := get(h, "/health", nil)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))

	other := get(h, "/health", nil)
	assert.Equal(t, http.StatusTooManyRequests, other.Code)
}

func TestRateLimit_IgnoresForwardedForUnlessTrusted(t *testing.T) {
	h := newTestServer(Options{RateLimit: 1})
	require.Equal(t, http.StatusOK, get(h, "/health", nil).Code)
	rec := get(h, "/health", map[string]string{"X-Forwarded-For": "198.51.100.1"})
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "a fake header must not reset the limit")

	h = newTestServer(Options{RateLimit: 1, TrustProxy: true})
	require.Equal(t, http.StatusOK, get(h, "/health", map[string]string{"X-Forwarded-For": "198.51.100.1"}).Code)
	assert.Equal(t, http.StatusOK, get(h, "/health", map[string]string{"X-Forwarded-For": "198.51.100.2"}).Code,
		"behind a trusted proxy each client IP has its own limit")
}

func TestRateLimit_IPv6CountsTheWholeSlash64(t *testing.T) {
	h := newTestServer(Options{RateLimit: 1, TrustProxy: true})
	require.Equal(t, http.StatusOK, get(h, "/health", map[string]string{"X-Forwarded-For": "2001:db8:1:2::1"}).Code)
	rec := get(h, "/health", map[string]string{"X-Forwarded-For": "2001:db8:1:2:ffff::9"})
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "another address of the same /64 is the same client")
	assert.Equal(t, http.StatusOK, get(h, "/health", map[string]string{"X-Forwarded-For": "2001:db8:1:3::1"}).Code)
}

func TestLoginRateLimit(t *testing.T) {
	h := newTestServer(Options{LoginRateLimit: 2})
	login := func() int {
		req := httptest.NewRequest(http.MethodPost, "/auth/login",
			strings.NewReader(`{"email":"x@example.com","password":"whatever-123"}`))
		req.RemoteAddr = "203.0.113.9:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.Equal(t, http.StatusUnauthorized, login())
	assert.Equal(t, http.StatusUnauthorized, login())
	assert.Equal(t, http.StatusTooManyRequests, login())
}

func TestTrustedProxy_UsesRightmostForwardedFor(t *testing.T) {
	h := newTestServer(Options{RateLimit: 1, TrustProxy: true})
	// The load balancer appends the real client IP at the end; anything to
	// its left was sent by the client and must not change the bucket.
	require.Equal(t, http.StatusOK, get(h, "/health", map[string]string{"X-Forwarded-For": "1.1.1.1, 198.51.100.7"}).Code)
	rec := get(h, "/health", map[string]string{"X-Forwarded-For": "2.2.2.2, 198.51.100.7"})
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "forged leftmost entries do not reset the limit")
}

func TestLastForwardedFor(t *testing.T) {
	cases := map[string]string{
		"":                      "",
		"203.0.113.1":           "203.0.113.1",
		"10.0.0.1, 203.0.113.1": "203.0.113.1",
		"10.0.0.1, not-an-ip":   "",
		" 2001:db8::1 ":         "2001:db8::1",
	}
	for header, want := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			req.Header.Set("X-Forwarded-For", header)
		}
		assert.Equal(t, want, lastForwardedFor(req), header)
	}
}
