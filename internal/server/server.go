// Package server wires the HTTP routes.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/Victor-Novakoski/rastreia/api"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/httpx"
	"github.com/Victor-Novakoski/rastreia/internal/push"
	"github.com/Victor-Novakoski/rastreia/internal/user"
)

type Deps struct {
	Tokens     *auth.Tokens
	Auth       *auth.Handler
	Users      *user.Handler
	Deliveries *delivery.Handler
	// Live serves the WebSocket routes; nil leaves them out.
	Live *delivery.LiveHandler
	// Push serves the Web Push routes; nil (no VAPID key) leaves them out.
	Push    *push.Handler
	Ready   func(r *http.Request) error
	Options Options
}

type Options struct {
	Production  bool
	CORSOrigins []string
	TrustProxy  bool
	// Redis shares the rate limits between API instances; nil counts in memory.
	Redis redis.UniversalClient
	// Requests per minute per IP. Zero uses the defaults.
	RateLimit         int
	LoginRateLimit    int
	TrackingRateLimit int
}

func New(d Deps) http.Handler {
	opts := d.Options
	if opts.RateLimit == 0 {
		opts.RateLimit = 120
	}
	if opts.LoginRateLimit == 0 {
		opts.LoginRateLimit = 10
	}
	if opts.TrackingRateLimit == 0 {
		opts.TrackingRateLimit = 30
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if opts.TrustProxy {
		r.Use(trustedProxy)
	}
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Use(securityHeaders(opts.Production), corsPolicy(opts.CORSOrigins))
	limit := func(name string, requests int) func(http.Handler) http.Handler {
		return rateLimit(opts.Redis, name, requests)
	}
	r.Use(limit("global", opts.RateLimit))
	// One limit for the page and its WebSocket: both answer whether a code exists.
	tracking := limit("tracking", opts.TrackingRateLimit)

	// WebSockets stay open, so they skip the request timeout below.
	if d.Live != nil {
		r.With(tracking).Get("/public/tracking/{code}/live", d.Live.Track)
		r.Get("/live/deliveries", d.Live.Panel)
	}

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(15 * time.Second))

		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			if err := d.Ready(r); err != nil {
				slog.Error("health", "err", err)
				httpx.Error(w, http.StatusServiceUnavailable, "dependency unavailable")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(api.OpenAPI)
		})

		r.With(limit("login", opts.LoginRateLimit)).Post("/auth/login", d.Auth.Login)
		r.With(limit("refresh", opts.LoginRateLimit)).Post("/auth/refresh", d.Auth.Refresh)
		r.With(limit("logout", opts.LoginRateLimit)).Post("/auth/logout", d.Auth.Logout)
		// Its own, tighter limit makes guessing tracking codes slow.
		r.With(tracking).Get("/public/tracking/{code}", d.Deliveries.Track)
		if d.Push != nil {
			r.Get("/public/push/key", d.Push.Key)
			r.With(tracking).Post("/public/tracking/{code}/push", d.Push.Subscribe)
			r.With(tracking).Delete("/public/tracking/{code}/push", d.Push.Unsubscribe)
		}

		r.Group(func(r chi.Router) {
			r.Use(d.Tokens.Authenticate, auth.RequireRole(auth.RoleAdmin))

			r.Get("/drivers", d.Users.ListDrivers)
			r.Post("/drivers", d.Users.CreateDriver)

			r.Get("/deliveries", d.Deliveries.List)
			r.Post("/deliveries", d.Deliveries.Create)
			r.Get("/deliveries/{id}", d.Deliveries.Get)
			r.Patch("/deliveries/{id}", d.Deliveries.Update)
		})

		// Drivers reach only their own deliveries here; the service answers 404
		// for anyone else's.
		r.Group(func(r chi.Router) {
			r.Use(d.Tokens.Authenticate, auth.RequireRole(auth.RoleAdmin, auth.RoleDriver))

			r.Get("/deliveries/{id}/events", d.Deliveries.ListEvents)
			r.Post("/deliveries/{id}/events", d.Deliveries.AddEvent)
		})

		r.Group(func(r chi.Router) {
			r.Use(d.Tokens.Authenticate, auth.RequireRole(auth.RoleDriver))

			r.Get("/me/deliveries", d.Deliveries.ListMine)
		})
	})

	return r
}
