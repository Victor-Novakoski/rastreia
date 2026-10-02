package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	httprateredis "github.com/go-chi/httprate-redis"
	"github.com/redis/go-redis/v9"

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
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders: []string{"Authorization", "Content-Type", "Idempotency-Key"},
		ExposedHeaders: []string{"Retry-After", "Idempotent-Replayed"},
		// The refresh token travels in a cookie (SECURITY.md #14 and #18).
		AllowCredentials: true,
		MaxAge:           600,
	})
}

// rateLimit allows requests per minute per client IP, taken from
// r.RemoteAddr (already rewritten by trustedProxy when the API is behind one).
// With a Redis client the count is shared by every API instance, and name
// keeps each limit's keys apart. If Redis stops answering, each instance
// counts in memory until it is back.
func rateLimit(rdb redis.UniversalClient, name string, requests int) func(http.Handler) http.Handler {
	opts := []httprate.Option{
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			httpx.Error(w, http.StatusTooManyRequests, "too many requests, try again later")
		}),
	}
	if rdb != nil {
		opts = append(opts, httprateredis.WithRedisLimitCounter(&httprateredis.Config{
			Client:       rdb,
			PrefixKey:    "rastreia:rate:" + name,
			WindowLength: time.Minute,
			OnFallbackChange: func(activated bool) {
				slog.Warn("rate limit fallback to memory", "limit", name, "active", activated)
			},
		}))
	}
	return httprate.LimitBy(requests, time.Minute, remoteIP, opts...)
}
