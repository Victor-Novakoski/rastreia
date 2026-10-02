package server

import (
	"net/http"
	"time"

	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/Victor-Novakoski/rastreia/internal/httpx"
)

// securityHeaders sets headers that keep browsers from sniffing, framing or
// caching API responses. HSTS only goes out in production, where the API is
// served over HTTPS.
func securityHeaders(production bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			h.Set("Cache-Control", "no-store")
			if production {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func corsPolicy(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodOptions},
		AllowedHeaders: []string{"Authorization", "Content-Type", "Idempotency-Key"},
		ExposedHeaders: []string{"Retry-After", "Idempotent-Replayed"},
		// The refresh token travels in a cookie (SECURITY.md #14 and #18).
		AllowCredentials: true,
		MaxAge:           600,
	})
}

// rateLimit allows requests per minute per client IP, taken from
// r.RemoteAddr (already rewritten by trustedProxy when the API is behind one).
func rateLimit(requests int) func(http.Handler) http.Handler {
	return httprate.LimitBy(requests, time.Minute, remoteIP,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			httpx.Error(w, http.StatusTooManyRequests, "too many requests, try again later")
		}),
	)
}
