package notify

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

// attempts reads the x-death header the way RabbitMQ writes it: one entry
// per queue the message was dead-lettered from, with a count.
func TestAttempts(t *testing.T) {
	d := amqp.Delivery{Headers: amqp.Table{"x-death": []any{
		amqp.Table{"queue": EmailQueue + ".retry", "reason": "expired", "count": int64(2)},
		amqp.Table{"queue": EmailQueue, "reason": "rejected", "count": int64(2)},
	}}}
	assert.Equal(t, int64(2), attempts(d, EmailQueue))
	assert.Zero(t, attempts(d, PushQueue))
	assert.Zero(t, attempts(amqp.Delivery{}, EmailQueue), "first try")
}
