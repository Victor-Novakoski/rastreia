package notify

import (
	"context"
	"errors"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Names in RabbitMQ. Every consumer of status changes gets its own queue
// bound to the exchange, so a new channel (push, SMS) does not touch e-mail.
const (
	Exchange   = "rastreia.events"
	RoutingKey = "delivery.status_changed"

	EmailQueue = "notifications.email"
	// A failed e-mail waits in EmailRetryQueue for RetryDelay and goes back
	// to EmailQueue; after MaxAttempts it is parked in EmailDeadQueue.
	EmailRetryQueue = "notifications.email.retry"
	EmailDeadQueue  = "notifications.email.dead"

	RetryDelayMillis = 30_000
)

// Declare creates the exchange and queues if they do not exist. Both the API
// and the worker call it, so messages published before the worker ever ran
// are kept instead of dropped by an exchange with no queue.
func Declare(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(Exchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return err
	}
	queues := []struct {
		name string
		args amqp.Table
	}{
		{EmailQueue, amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": EmailRetryQueue,
		}},
		{EmailRetryQueue, amqp.Table{
			"x-message-ttl":             int32(RetryDelayMillis),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": EmailQueue,
		}},
		{EmailDeadQueue, nil},
	}
	for _, q := range queues {
		if _, err := ch.QueueDeclare(q.name, true, false, false, false, q.args); err != nil {
			return fmt.Errorf("queue %s: %w", q.name, err)
		}
	}
	return ch.QueueBind(EmailQueue, RoutingKey, Exchange, false, nil)
}

// Publisher sends messages with publisher confirms: Publish returns only once
// RabbitMQ has stored the message, so the outbox row can be marked as sent.
// It reconnects on the next call after the connection drops.
type Publisher struct {
	mu   sync.Mutex
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(url string) *Publisher {
	return &Publisher{url: url}
}

func (p *Publisher) channel() (*amqp.Channel, error) {
	if p.ch != nil && !p.ch.IsClosed() {
		return p.ch, nil
	}
	p.close()
	conn, err := amqp.Dial(p.url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err == nil {
		err = Declare(ch)
	}
	if err == nil {
		err = ch.Confirm(false)
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	p.conn, p.ch = conn, ch
	return ch, nil
}

// Publish sends one persistent message to the events exchange.
func (p *Publisher) Publish(ctx context.Context, msgID string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	ch, err := p.channel()
	if err != nil {
		return err
	}
	conf, err := ch.PublishWithDeferredConfirmWithContext(ctx, Exchange, RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    msgID,
		Body:         body,
	})
	if err != nil {
		return err
	}
	ok, err := conf.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("rabbitmq refused the message")
	}
	return nil
}

func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.close()
}

func (p *Publisher) close() {
	if p.conn != nil {
		_ = p.conn.Close()
	}
	p.conn, p.ch = nil, nil
}
