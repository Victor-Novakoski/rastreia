package retention_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/retention"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/testdb"
)

func TestPG_AnonymizesOldFinishedDeliveries(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	q := store.New(pool)
	carrierID := testdb.Carrier(t, pool)
	u, err := q.CreateUser(ctx, store.CreateUserParams{CarrierID: carrierID, Name: "Dona", Email: "dona@example.com", PasswordHash: "x", Role: auth.RoleCarrier})
	require.NoError(t, err)
	owner := auth.Claims{UserID: u.ID, Role: auth.RoleCarrier, CarrierID: carrierID}
	svc := delivery.NewService(delivery.NewPGStore(pool))
	create := func() delivery.Delivery {
		d, err := svc.Create(ctx, owner, delivery.CreateInput{
			RecipientName: "Maria Souza", RecipientEmail: "maria@example.com", Address: "Rua A, 10",
		})
		require.NoError(t, err)
		return d
	}
	move := func(d delivery.Delivery, statuses ...string) {
		for _, s := range statuses {
			in := delivery.EventInput{Status: s}
			if s == delivery.StatusFailed {
				note := "vizinho do 32 não quis receber"
				in.Note = &note
			}
			_, err := svc.AddEvent(ctx, owner, d.ID, in)
			require.NoError(t, err)
		}
	}

	old := create()
	move(old, delivery.StatusPickedUp, delivery.StatusInTransit, delivery.StatusFailed)
	recent := create()
	move(recent, delivery.StatusPickedUp, delivery.StatusInTransit, delivery.StatusDelivered)
	open := create()
	_, err = pool.Exec(ctx, "UPDATE deliveries SET completed_at = now() - interval '100 days' WHERE id = $1", old.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE deliveries SET created_at = now() - interval '200 days' WHERE id = $1", open.ID)
	require.NoError(t, err)

	job := retention.NewJob(q, 90*24*time.Hour)
	n, err := job.Once(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	got, err := svc.Get(ctx, owner, old.ID)
	require.NoError(t, err)
	assert.Equal(t, "Destinatário removido", got.RecipientName)
	assert.Empty(t, got.RecipientEmail)
	assert.Empty(t, got.Address)
	assert.NotNil(t, got.AnonymizedAt)
	events, err := svc.ListEvents(ctx, owner, old.ID)
	require.NoError(t, err)
	assert.Len(t, events, 4, "history stays")
	for _, e := range events {
		assert.Nil(t, e.Note)
	}

	for _, id := range []int64{recent.ID, open.ID} {
		d, err := svc.Get(ctx, owner, id)
		require.NoError(t, err)
		assert.Equal(t, "Maria Souza", d.RecipientName)
	}

	// Erased deliveries no longer change, or the recipient's data could come back.
	_, err = svc.AddEvent(ctx, owner, old.ID, delivery.EventInput{Status: delivery.StatusInTransit})
	assert.ErrorIs(t, err, apperr.ErrConflict)
	name := "Outra Pessoa"
	_, err = svc.Update(ctx, owner, old.ID, delivery.UpdateInput{RecipientName: &name})
	assert.ErrorIs(t, err, apperr.ErrConflict)

	n, err = job.Once(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "runs again without touching it")
}
