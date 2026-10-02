// Command worker sends the notification e-mails queued by the API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Victor-Novakoski/rastreia/internal/config"
	"github.com/Victor-Novakoski/rastreia/internal/notify"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.LoadWorker()
	if err != nil {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mailer := notify.NewSMTP(notify.SMTPConfig{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort,
		Username: cfg.SMTPUsername, Password: cfg.SMTPPassword,
		From: cfg.SMTPFrom, TLS: cfg.SMTPTLS,
	})
	if err := notify.NewWorker(cfg.RabbitMQURL, mailer, cfg.TrackingURL).Run(ctx); err != nil {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}
