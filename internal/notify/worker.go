package notify

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Handler delivers one status change on one channel (e-mail, push). An
// ErrBadMessage parks the message; any other error retries it.
type Handler func(ctx context.Context, m StatusChanged) error

// Consumer reads one queue.
type Consumer struct {
	Queue  string
	Handle Handler
	// MaxAttempts counts the first try; after that the message is parked
	// in the dead queue for someone to look at.
	MaxAttempts int64
	// MaxAge drops messages about changes older than this without handling
	// them, like the relay does before publishing: after the worker was
	// down for a night, a pile of old statuses only confuses the recipient.
	MaxAge time.Duration
}

// DiscardConsumer empties a queue nobody handles. Every status change is
// routed to the push queue too, so without VAPID keys the worker still
// reads it, or the messages would pile up in RabbitMQ.
func DiscardConsumer(queue string) Consumer {
	return Consumer{Queue: queue, Handle: func(context.Context, StatusChanged) error { return nil }}
}

// Worker runs consumers on one RabbitMQ connection.
type Worker struct {
	url       string
	consumers []Consumer
}

func NewWorker(url string, consumers ...Consumer) *Worker {
	for i := range consumers {
		if consumers[i].MaxAttempts == 0 {
			consumers[i].MaxAttempts = 5
		}
		if consumers[i].MaxAge == 0 {
			consumers[i].MaxAge = time.Hour
		}
	}
	return &Worker{url: url, consumers: consumers}
}

// Run consumes until ctx is done, reconnecting when RabbitMQ goes away.
func (w *Worker) Run(ctx context.Context) error {
	backoff := time.Second
	for {
		started := time.Now()
		err := w.consume(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if time.Since(started) > time.Minute {
			backoff = time.Second // it ran for a while: a new outage starts over
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errs := make(chan error, len(w.consumers))
	for _, c := range w.consumers {
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
		msgs, err := ch.ConsumeWithContext(ctx, c.Queue, "", false, false, false, false, nil)
		if err != nil {
			return err
		}
		slog.Info("worker consuming", "queue", c.Queue)
		go func() { errs <- c.loop(ctx, ch, msgs) }()
	}
	// One broken channel restarts the whole connection.
	select {
	case <-ctx.Done():
		return nil
	case err := <-errs:
		return err
	}
}

func (c Consumer) loop(ctx context.Context, ch *amqp.Channel, msgs <-chan amqp.Delivery) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return errors.New("channel closed")
			}
			if err := c.handle(ctx, ch, d); err != nil {
				return err
			}
		}
	}
}

// handle runs the handler and settles the message. Only an error talking to
// RabbitMQ is returned; a failed delivery is retried or parked.
func (c Consumer) handle(ctx context.Context, ch *amqp.Channel, d amqp.Delivery) error {
	var m StatusChanged
	err := json.Unmarshal(d.Body, &m)
	switch {
	case err != nil:
		err = ErrBadMessage{err}
	case c.MaxAge > 0 && time.Since(m.OccurredAt) > c.MaxAge:
		slog.Info("notification dropped, too old", "queue", c.Queue, "message_id", d.MessageId, "occurred_at", m.OccurredAt)
		return d.Ack(false)
	default:
		err = c.Handle(ctx, m)
	}
	if err == nil {
		return d.Ack(false)
	}
	attempt := attempts(d, c.Queue) + 1
	var bad ErrBadMessage
	if errors.As(err, &bad) || attempt >= c.MaxAttempts {
		slog.Error("notification parked", "queue", c.Queue, "message_id", d.MessageId, "attempt", attempt, "err", err)
		if err := ch.PublishWithContext(ctx, "", deadQueue(c.Queue), false, false, amqp.Publishing{
			ContentType: d.ContentType, DeliveryMode: amqp.Persistent,
			MessageId: d.MessageId, Body: d.Body,
			Headers: amqp.Table{"x-error": err.Error()},
		}); err != nil {
			return err
		}
		return d.Ack(false)
	}
	slog.Warn("notification failed, will retry", "queue", c.Queue, "message_id", d.MessageId, "attempt", attempt, "err", err)
	// Rejected without requeue, the message dead-letters to the retry queue.
	return d.Reject(false)
}

// attempts reads how many times RabbitMQ dead-lettered the message out of
// queue, which is how many tries already failed.
func attempts(d amqp.Delivery, queue string) int64 {
	deaths, _ := d.Headers["x-death"].([]any)
	for _, x := range deaths {
		t, _ := x.(amqp.Table)
		if t["queue"] == queue {
			n, _ := t["count"].(int64)
			return n
		}
	}
	return 0
}
