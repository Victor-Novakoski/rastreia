package auth

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/Victor-Novakoski/rastreia/internal/httpx"
)

type ctxKey struct{}

// FromContext returns the claims set by Authenticate.
func FromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(Claims)
	return c, ok
}

func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// Authenticate requires a valid "Authorization: Bearer <token>" header. The
// request's log line gets who made it, or why it was refused.
func (t *Tokens) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || raw == "" {
			httpx.AddLogAttrs(r.Context(), slog.String("reason", "missing token"))
			httpx.Error(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		claims, err := t.Parse(raw)
		if err != nil {
			httpx.AddLogAttrs(r.Context(), slog.String("reason", "invalid token"))
			httpx.Error(w, http.StatusUnauthorized, "invalid token")
			return
		}
		httpx.AddLogAttrs(r.Context(),
			slog.Int64("user_id", claims.UserID), slog.Int64("carrier_id", claims.CarrierID))
		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
	})
}

// RequireRole lets the request through only for the given roles. It must run
// after Authenticate.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			if !slices.Contains(roles, claims.Role) {
				httpx.AddLogAttrs(r.Context(), slog.String("reason", "role "+claims.Role+" not allowed"))
				httpx.Error(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
