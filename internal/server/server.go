// Package server wires the HTTP routes.
package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Victor-Novakoski/rastreia/api"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/httpx"
	"github.com/Victor-Novakoski/rastreia/internal/user"
)

type Deps struct {
	Tokens     *auth.Tokens
	Auth       *auth.Handler
	Users      *user.Handler
	Deliveries *delivery.Handler
	Ready      func(r *http.Request) error
	Options    Options
}

type Options struct {
	Production  bool
	CORSOrigins []string
	TrustProxy  bool
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
	r.Use(rateLimit(opts.RateLimit))
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := d.Ready(r); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(api.OpenAPI)
	})

	r.With(rateLimit(opts.LoginRateLimit)).Post("/auth/login", d.Auth.Login)
	// Its own, tighter limit makes guessing tracking codes slow.
	r.With(rateLimit(opts.TrackingRateLimit)).Get("/public/tracking/{code}", d.Deliveries.Track)

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

	return r
}
