package auth

import (
	"context"
	"encoding/json"
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
	users := fakeUsers{"ana@example.com": {ID: 7, Email: "ana@example.com", PasswordHash: hash, Role: RoleDriver}}
	tokens := NewTokens(testSecret, time.Hour)
	h := NewHandler(users, tokens, NewLoginGuard())

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
				claims, err := tokens.Parse(resp.Token)
				require.NoError(t, err)
				assert.Equal(t, int64(7), claims.UserID)
			}
		})
	}
}

func TestLogin_LocksAfterRepeatedFailures(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	users := fakeUsers{"ana@example.com": {ID: 7, Email: "ana@example.com", PasswordHash: hash, Role: RoleDriver}}
	h := NewHandler(users, NewTokens(testSecret, time.Hour), NewLoginGuard())

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
