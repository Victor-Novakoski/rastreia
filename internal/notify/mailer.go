package notify

import (
	"context"
	"fmt"
	"log/slog"
)

// Mailer sends one e-mail. *SMTP implements it.
type Mailer interface {
	Send(ctx context.Context, e Email) error
}

// EmailConsumer e-mails the recipient about each status change.
func EmailConsumer(mailer Mailer, trackingURL string) Consumer {
	return Consumer{Queue: EmailQueue, Handle: func(ctx context.Context, m StatusChanged) error {
		e, err := Render(m, trackingURL)
		if err != nil {
			return ErrBadMessage{err}
		}
		if err := mailer.Send(ctx, e); err != nil {
			return fmt.Errorf("send: %w", err)
		}
		slog.Info("e-mail sent", "event_id", m.EventID, "status", m.Status)
		return nil
	}}
}
