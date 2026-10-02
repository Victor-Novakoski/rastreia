package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/config"
	"github.com/Victor-Novakoski/rastreia/internal/database"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/realtime"
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
	// Without Redis everything stays in memory, which is right for one instance.
	var (
		rdb    redis.UniversalClient
		broker realtime.Broker = realtime.NewLocal()
		guard  auth.Guard      = auth.NewLoginGuard()
	)
	if cfg.RedisURL != "" {
		opts, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return fmt.Errorf("REDIS_URL: %w", err)
		}
		client := redis.NewClient(opts)
		defer func() { _ = client.Close() }()
		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		if broker, err = realtime.NewRedis(ctx, client); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		rdb, guard = client, auth.NewRedisGuard(client)
	}
	deliveries := delivery.NewService(delivery.NewPGStore(pool)).WithPublisher(broker)
	live := realtime.NewServer(ctx, broker, realtime.Options{Origins: cfg.AllowedOrigins()})

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
			Tokens: tokens,
			Auth: auth.NewHandler(queries, tokens, guard,
				auth.NewSessions(queries, cfg.RefreshTTL),
				auth.CookieOptions{AllowedOrigins: cfg.AllowedOrigins()}),
			Users:      user.NewHandler(users),
			Deliveries: delivery.NewHandler(deliveries),
			Live:       delivery.NewLiveHandler(deliveries, live, tokens),
			Ready: func(r *http.Request) error {
				if err := pool.Ping(r.Context()); err != nil {
					return fmt.Errorf("database: %w", err)
				}
				if rdb != nil {
					if err := rdb.Ping(r.Context()).Err(); err != nil {
						return fmt.Errorf("redis: %w", err)
					}
				}
				return nil
			},
			Options: server.Options{
				Production:  cfg.IsProduction(),
				CORSOrigins: cfg.AllowedOrigins(),
				TrustProxy:  cfg.TrustProxy,
				Redis:       rdb,
			},
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
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
