// Package notify tells recipients when their delivery changes status. The API
// records each change in the outbox (delivery_events), the Relay publishes it
// to RabbitMQ, and the worker (cmd/worker) reads the queues and sends the
// e-mail and the Web Push notifications.
package notify

import (
	"time"
)

// StatusChanged is the message published for every delivery event. It carries
// the recipient's name and e-mail, so sending the e-mail needs no database
// (only Web Push reads the subscriptions from it).
type StatusChanged struct {
	EventID        int64     `json:"event_id"`
	TrackingCode   string    `json:"tracking_code"`
	Status         string    `json:"status"`
	RecipientName  string    `json:"recipient_name"`
	RecipientEmail string    `json:"recipient_email"`
	OccurredAt     time.Time `json:"occurred_at"`
}
