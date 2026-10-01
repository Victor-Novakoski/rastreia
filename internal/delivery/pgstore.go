package delivery

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// PGStore is the Store backed by PostgreSQL.
type PGStore struct {
	*store.Queries
	pool *pgxpool.Pool
	inTx bool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{Queries: store.New(pool), pool: pool}
}

// InTx commits when fn returns nil and rolls back otherwise. Called on a
// store that is already in a transaction, it reuses that transaction.
func (p *PGStore) InTx(ctx context.Context, fn func(Store) error) error {
	if p.inTx {
		return fn(p)
	}
	return pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		return fn(&PGStore{Queries: p.WithTx(tx), pool: p.pool, inTx: true})
	})
}
