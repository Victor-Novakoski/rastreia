// Package retention erases the recipient's personal data some time after a
// delivery is finished (LGPD, SECURITY.md #27), and deletes the rows that
// only mattered for a while: old idempotency keys and expired refresh tokens.
package retention

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type Store interface {
	AnonymizeDeliveries(ctx context.Context, arg store.AnonymizeDeliveriesParams) (int64, error)
	DeleteOldIdempotencyKeys(ctx context.Context) (int64, error)
	DeleteExpiredRefreshTokens(ctx context.Context) (int64, error)
}

// Job anonymizes deliveries finished more than Keep ago, and the ones never
// finished created more than Abandoned ago, and cleans up. The statements only
// touch rows not handled yet, so several API instances can run it.
type Job struct {
	store     Store
	Keep      time.Duration
	Abandoned time.Duration
	Every     time.Duration
	now       func() time.Time
}

// abandoned is how long a delivery nobody finished keeps the recipient's
// data: after a year it is not going to be delivered.
const abandoned = 365 * 24 * time.Hour

func NewJob(s Store, keep time.Duration) *Job {
	return &Job{store: s, Keep: keep, Abandoned: abandoned, Every: time.Hour, now: time.Now}
}

// Run anonymizes once now and then every j.Every until ctx is done.
func (j *Job) Run(ctx context.Context) {
	t := time.NewTicker(j.Every)
	defer t.Stop()
	for {
		if _, err := j.Once(ctx); err != nil && ctx.Err() == nil {
			slog.Error("retention", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Once runs every step, even when one fails, and returns how many
// deliveries were anonymized.
func (j *Job) Once(ctx context.Context) (int64, error) {
	now := j.now()
	n, err := j.store.AnonymizeDeliveries(ctx, store.AnonymizeDeliveriesParams{
		Before: now.Add(-j.Keep), AbandonedBefore: now.Add(-j.Abandoned),
	})
	if n > 0 {
		slog.Info("retention anonymized deliveries", "count", n)
	}
	keys, kerr := j.store.DeleteOldIdempotencyKeys(ctx)
	tokens, terr := j.store.DeleteExpiredRefreshTokens(ctx)
	if keys > 0 || tokens > 0 {
		slog.Info("retention deleted expired rows", "idempotency_keys", keys, "refresh_tokens", tokens)
	}
	return n, errors.Join(err, kerr, terr)
}

var _ Store = (*store.Queries)(nil)
