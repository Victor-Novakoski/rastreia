package notify

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// Sender publishes one message. *Publisher implements it.
type Sender interface {
	Publish(ctx context.Context, msgID string, body []byte) error
}

// Relay moves events from the outbox to RabbitMQ. If RabbitMQ is down the
// events stay in the database and go out when it is back, so the API never
// fails a status change because of a notification.
type Relay struct {
	pool     *pgxpool.Pool
	sender   Sender
	Interval time.Duration
	Batch    int32
	// MaxAge drops events older than this: an e-mail about a status from
	// yesterday only confuses the recipient.
	MaxAge time.Duration
	now    func() time.Time
}

func NewRelay(pool *pgxpool.Pool, sender Sender) *Relay {
	return &Relay{pool: pool, sender: sender, Interval: time.Second, Batch: 100, MaxAge: time.Hour, now: time.Now}
}

// Run publishes pending events until ctx is done.
func (r *Relay) Run(ctx context.Context) {
	t := time.NewTicker(r.Interval)
	defer t.Stop()
	for {
		for {
			n, err := r.Flush(ctx)
			if err != nil {
				if ctx.Err() == nil {
					slog.Error("notify relay", "err", err)
				}
				break
			}
			if n < int(r.Batch) {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Flush publishes one batch and returns how many events went out. The rows
// stay locked until the commit, so another instance skips them; if a publish
// fails, the events already sent are marked and the rest wait for the next
// try. A crash between publish and commit sends an event twice, so delivery
// is at least once.
func (r *Relay) Flush(ctx context.Context) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	skipped, err := q.SkipStaleEvents(ctx, r.now().Add(-r.MaxAge))
	if err != nil {
		return 0, err
	}
	if skipped > 0 {
		slog.Warn("notify relay skipped stale events", "count", skipped)
	}
	rows, err := q.ClaimUnpublishedEvents(ctx, r.Batch)
	if err != nil {
		return 0, err
	}
	ids := make([]int64, 0, len(rows))
	var pubErr error
	for _, row := range rows {
		body, err := json.Marshal(StatusChanged{
			EventID:        row.ID,
			TrackingCode:   row.TrackingCode,
			Status:         row.Status,
			RecipientName:  row.RecipientName,
			RecipientEmail: row.RecipientEmail,
			OccurredAt:     row.CreatedAt,
		})
		if err != nil {
			return 0, err
		}
		if pubErr = r.sender.Publish(ctx, strconv.FormatInt(row.ID, 10), body); pubErr != nil {
			break
		}
		ids = append(ids, row.ID)
	}
	if len(ids) > 0 {
		if err := q.MarkEventsPublished(ctx, ids); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(ids), pubErr
}
