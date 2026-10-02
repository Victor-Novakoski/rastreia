package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Mailer sends one e-mail. *SMTP implements it.
type Mailer interface {
	Send(ctx context.Context, e Email) error
}

// Worker reads status changes from EmailQueue and e-mails the recipient.
type Worker struct {
	url         string
	mailer      Mailer
	trackingURL string
	// MaxAttempts counts the first try; after that the message is parked
	// in EmailDeadQueue for someone to look at.
	MaxAttempts int64
}

func NewWorker(url string, mailer Mailer, trackingURL string) *Worker {
	return &Worker{url: url, mailer: mailer, trackingURL: trackingURL, MaxAttempts: 5}
}

// Run consumes until ctx is done, reconnecting when RabbitMQ goes away.
func (w *Worker) Run(ctx context.Context) error {
	backoff := time.Second
	for {
		err := w.consume(ctx)
		if ctx.Err() != nil {
			return nil
		}
		slog.Error("worker disconnected", "err", err, "retry_in", backoff.String())
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

func (w *Worker) consume(ctx context.Context) error {
	conn, err := amqp.Dial(w.url)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := Declare(ch); err != nil {
		return err
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return err
	}
	msgs, err := ch.ConsumeWithContext(ctx, EmailQueue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	slog.Info("worker consuming", "queue", EmailQueue)
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return errors.New("channel closed")
			}
			if err := w.handle(ctx, ch, d); err != nil {
				return err
			}
		}
	}
}

// handle sends one e-mail and settles the message. Only an error talking to
// RabbitMQ is returned; a failed e-mail is retried or parked.
func (w *Worker) handle(ctx context.Context, ch *amqp.Channel, d amqp.Delivery) error {
	err := w.process(ctx, d.Body)
	if err == nil {
		return d.Ack(false)
	}
	attempt := attempts(d) + 1
	var bad ErrBadMessage
	if errors.As(err, &bad) || attempt >= w.MaxAttempts {
		slog.Error("e-mail parked", "message_id", d.MessageId, "attempt", attempt, "err", err)
		if err := ch.PublishWithContext(ctx, "", EmailDeadQueue, false, false, amqp.Publishing{
			ContentType: d.ContentType, DeliveryMode: amqp.Persistent,
			MessageId: d.MessageId, Body: d.Body,
			Headers: amqp.Table{"x-error": err.Error()},
		}); err != nil {
			return err
		}
		return d.Ack(false)
	}
	slog.Warn("e-mail failed, will retry", "message_id", d.MessageId, "attempt", attempt, "err", err)
	// Rejected without requeue, the message dead-letters to the retry queue.
	return d.Reject(false)
}

func (w *Worker) process(ctx context.Context, body []byte) error {
	var m StatusChanged
	if err := json.Unmarshal(body, &m); err != nil {
		return ErrBadMessage{err}
	}
	e, err := Render(m, w.trackingURL)
	if err != nil {
		return ErrBadMessage{err}
	}
	if err := w.mailer.Send(ctx, e); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	slog.Info("e-mail sent", "event_id", m.EventID, "status", m.Status)
	return nil
}

// attempts reads how many times RabbitMQ dead-lettered the message out of
// EmailQueue, which is how many tries already failed.
func attempts(d amqp.Delivery) int64 {
	deaths, _ := d.Headers["x-death"].([]any)
	for _, x := range deaths {
		t, _ := x.(amqp.Table)
		if t["queue"] == EmailQueue {
			n, _ := t["count"].(int64)
			return n
		}
	}
	return 0
}
