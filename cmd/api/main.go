package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/config"
	"github.com/Victor-Novakoski/rastreia/internal/database"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/server"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/user"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := database.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	queries := store.New(pool)
	tokens := auth.NewTokens(cfg.JWTSecret, cfg.JWTTTL)
	users := user.NewService(queries)

	if cfg.AdminEmail != "" {
		err := users.EnsureAdmin(ctx, user.CreateInput{
			Name: cfg.AdminName, Email: cfg.AdminEmail, Password: cfg.AdminPassword,
		})
		if err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: server.New(server.Deps{
			Tokens:     tokens,
			Auth:       auth.NewHandler(queries, tokens, auth.NewLoginGuard()),
			Users:      user.NewHandler(users),
			Deliveries: delivery.NewHandler(delivery.NewService(queries)),
			Ready:      func(r *http.Request) error { return pool.Ping(r.Context()) },
			Options: server.Options{
				Production:  cfg.IsProduction(),
				CORSOrigins: cfg.AllowedOrigins(),
				TrustProxy:  cfg.TrustProxy,
			},
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "port", cfg.Port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
