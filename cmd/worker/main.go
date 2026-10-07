// Command worker sends the notifications queued by the API: e-mails and,
// with VAPID keys, Web Push. "worker vapid" prints a new pair of keys.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/Victor-Novakoski/rastreia/internal/config"
	"github.com/Victor-Novakoski/rastreia/internal/database"
	"github.com/Victor-Novakoski/rastreia/internal/notify"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "vapid" {
		private, public, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("VAPID_PUBLIC_KEY=%s\nVAPID_PRIVATE_KEY=%s\n", public, private)
		return
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadWorker()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mailer := notify.NewSMTP(notify.SMTPConfig{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort,
		Username: cfg.SMTPUsername, Password: cfg.SMTPPassword,
		From: cfg.SMTPFrom, TLS: cfg.SMTPTLS,
	})
	consumers := []notify.Consumer{notify.EmailConsumer(mailer, cfg.TrackingURL)}
	if cfg.PushEnabled() {
		pool, err := database.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		consumers = append(consumers, notify.PushConsumer(store.New(pool), notify.WebPush(notify.VAPID{
			PublicKey: cfg.VAPIDPublicKey, PrivateKey: cfg.VAPIDPrivateKey, Subject: cfg.VAPIDSubject,
		}), cfg.TrackingURL))
	} else {
		consumers = append(consumers, notify.DiscardConsumer(notify.PushQueue))
	}
	return notify.NewWorker(cfg.RabbitMQURL, consumers...).Run(ctx)
}
