package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateAndRequireRole(t *testing.T) {
	tokens := NewTokens(testSecret, time.Hour)
	ownerToken, err := tokens.Issue(Claims{UserID: 1, Role: RoleCarrier, CarrierID: 1})
	require.NoError(t, err)
	driverToken, err := tokens.Issue(Claims{UserID: 2, Role: RoleDriver, CarrierID: 1})
	require.NoError(t, err)

	var seen Claims
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = FromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	handler := tokens.Authenticate(RequireRole(RoleCarrier)(ok))

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"not bearer", "Basic abc", http.StatusUnauthorized},
		{"invalid token", "Bearer nope", http.StatusUnauthorized},
		{"wrong role", "Bearer " + driverToken, http.StatusForbidden},
		{"admin", "Bearer " + ownerToken, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tc.want, rec.Code)
		})
	}
	assert.Equal(t, Claims{UserID: 1, Role: RoleCarrier, CarrierID: 1}, seen)
}
