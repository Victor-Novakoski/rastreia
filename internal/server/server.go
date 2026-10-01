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
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
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

	r.Post("/auth/login", d.Auth.Login)

	r.Group(func(r chi.Router) {
		r.Use(d.Tokens.Authenticate, auth.RequireRole(auth.RoleAdmin))

		r.Get("/drivers", d.Users.ListDrivers)
		r.Post("/drivers", d.Users.CreateDriver)

		r.Get("/deliveries", d.Deliveries.List)
		r.Post("/deliveries", d.Deliveries.Create)
		r.Get("/deliveries/{id}", d.Deliveries.Get)
		r.Patch("/deliveries/{id}", d.Deliveries.Update)
	})

	return r
}
