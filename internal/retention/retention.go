// Package retention erases the recipient's personal data some time after a
// delivery is finished (LGPD, SECURITY.md #27).
package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type Store interface {
	AnonymizeDeliveries(ctx context.Context, before time.Time) (int64, error)
}

// Job anonymizes deliveries finished more than Keep ago. The update only
// touches rows not anonymized yet, so several API instances can run it.
type Job struct {
	store Store
	Keep  time.Duration
	Every time.Duration
	now   func() time.Time
}

func NewJob(s Store, keep time.Duration) *Job {
	return &Job{store: s, Keep: keep, Every: time.Hour, now: time.Now}
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

func (j *Job) Once(ctx context.Context) (int64, error) {
	n, err := j.store.AnonymizeDeliveries(ctx, j.now().Add(-j.Keep))
	if n > 0 {
		slog.Info("retention anonymized deliveries", "count", n)
	}
	return n, err
}

var _ Store = (*store.Queries)(nil)
