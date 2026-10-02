package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type fakeUsers map[string]store.User

func (f fakeUsers) GetUserByEmail(_ context.Context, email string) (store.User, error) {
	u, ok := f[email]
	if !ok {
		return store.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func TestLogin(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	users := fakeUsers{"ana@example.com": {ID: 7, Email: "ana@example.com", PasswordHash: hash, Role: RoleDriver, CarrierID: 1}}
	tokens := NewTokens(testSecret, time.Hour)
	h := NewHandler(users, tokens, NewLoginGuard(), NewSessions(newFakeSessions(), time.Hour), CookieOptions{})

	cases := []struct {
		name string
		body string
		want int
	}{
		{"ok, e-mail is normalized", `{"email":" Ana@Example.com ","password":"correct-horse"}`, http.StatusOK},
		{"wrong password", `{"email":"ana@example.com","password":"nope"}`, http.StatusUnauthorized},
		{"unknown e-mail", `{"email":"bob@example.com","password":"correct-horse"}`, http.StatusUnauthorized},
		{"bad json", `{"email":`, http.StatusBadRequest},
		{"unknown field", `{"email":"ana@example.com","password":"x","admin":true}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.Login(rec, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(tc.body)))
			require.Equal(t, tc.want, rec.Code, rec.Body.String())

			if tc.want == http.StatusOK {
				var resp loginResponse
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, RoleDriver, resp.Role)
				assert.Equal(t, 3600, resp.ExpiresIn)
				claims, err := tokens.Parse(resp.Token)
				require.NoError(t, err)
				assert.Equal(t, int64(7), claims.UserID)
				cookie := refreshCookie(t, rec)
				assert.True(t, cookie.HttpOnly)
				assert.Equal(t, "/auth", cookie.Path)
				assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
			}
		})
	}
}

func TestLogin_LocksAfterRepeatedFailures(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	users := fakeUsers{"ana@example.com": {ID: 7, Email: "ana@example.com", PasswordHash: hash, Role: RoleDriver, CarrierID: 1}}
	h := NewHandler(users, NewTokens(testSecret, time.Hour), NewLoginGuard(), NewSessions(newFakeSessions(), time.Hour), CookieOptions{})

	login := func(password string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		body := `{"email":"ana@example.com","password":"` + password + `"}`
		h.Login(rec, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body)))
		return rec
	}

	for range 5 {
		require.Equal(t, http.StatusUnauthorized, login("wrong").Code)
	}
	rec := login("correct-horse")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "even the right password waits for the lock")
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

const testOrigin = "http://localhost:5173"

func refreshCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == RefreshCookie {
			return c
		}
	}
	t.Fatal("no refresh cookie")
	return nil
}

func TestRefreshAndLogout(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	users := fakeUsers{"ana@example.com": {ID: 7, Email: "ana@example.com", PasswordHash: hash, Role: RoleDriver, CarrierID: 1}}
	tokens := NewTokens(testSecret, time.Hour)
	h := NewHandler(users, tokens, NewLoginGuard(), NewSessions(newFakeSessions(), time.Hour),
		CookieOptions{AllowedOrigins: []string{testOrigin}})

	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"ana@example.com","password":"correct-horse"}`)))
	require.Equal(t, http.StatusOK, rec.Code)
	first := refreshCookie(t, rec)
	assert.True(t, first.Secure)

	post := func(handler http.HandlerFunc, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/auth/x", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}

	assert.Equal(t, http.StatusForbidden, post(h.Refresh, "", first).Code, "no Origin")
	assert.Equal(t, http.StatusForbidden, post(h.Refresh, "https://evil.example", first).Code, "foreign Origin")
	assert.Equal(t, http.StatusUnauthorized, post(h.Refresh, testOrigin, nil).Code, "no cookie")

	rec = post(h.Refresh, testOrigin, first)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp loginResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	claims, err := tokens.Parse(resp.Token)
	require.NoError(t, err)
	assert.Equal(t, Claims{UserID: 7, Role: RoleDriver, CarrierID: 1}, claims)
	second := refreshCookie(t, rec)
	assert.NotEqual(t, first.Value, second.Value)

	assert.Equal(t, http.StatusNoContent, post(h.Logout, testOrigin, second).Code)
	rec = post(h.Refresh, testOrigin, second)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "session ended")
	assert.Equal(t, -1, refreshCookie(t, rec).MaxAge, "invalid session clears the cookie")
}

type brokenGuard struct{ *LoginGuard }

func (brokenGuard) Check(context.Context, string) (time.Duration, error) {
	return 0, errors.New("redis down")
}

func TestLogin_WithoutGuardRefuses(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	users := fakeUsers{"ana@example.com": {ID: 7, Email: "ana@example.com", PasswordHash: hash, Role: RoleDriver, CarrierID: 1}}
	h := NewHandler(users, NewTokens(testSecret, time.Hour), brokenGuard{NewLoginGuard()}, NewSessions(newFakeSessions(), time.Hour), CookieOptions{})

	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"ana@example.com","password":"correct-horse"}`)))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "no login without brute-force protection")
}
