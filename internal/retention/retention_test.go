package retention_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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
			RecipientName: "Maria Souza", RecipientEmail: "maria@example.com", RecipientPhone: "11987654321",
			PostalCode: "01001000", Street: "Praça da Sé", Number: "10", District: "Sé", City: "São Paulo", State: "SP",
			Latitude: ptrTo(-23.55), Longitude: ptrTo(-46.63),
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
	abandoned := create()
	move(abandoned, delivery.StatusPickedUp)
	_, err = pool.Exec(ctx, "UPDATE deliveries SET completed_at = now() - interval '100 days' WHERE id = $1", old.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE deliveries SET created_at = now() - interval '200 days' WHERE id = $1", open.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE deliveries SET created_at = now() - interval '400 days' WHERE id = $1", abandoned.ID)
	require.NoError(t, err)
	follow := func(d delivery.Delivery) {
		err := q.UpsertPushSubscription(ctx, store.UpsertPushSubscriptionParams{
			DeliveryID: d.ID, Endpoint: "https://fcm.googleapis.com/fcm/send/" + d.TrackingCode, P256dh: "k", Auth: "a",
		})
		require.NoError(t, err)
	}
	follow(old)
	follow(open)

	job := retention.NewJob(q, 90*24*time.Hour)
	n, err := job.Once(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n, "the old finished one and the one nobody finished in a year")

	for _, id := range []int64{old.ID, abandoned.ID} {
		got, err := svc.Get(ctx, owner, id)
		require.NoError(t, err)
		assert.Equal(t, "Destinatário removido", got.RecipientName)
		assert.Empty(t, got.RecipientEmail)
		assert.Empty(t, got.Address)
		assert.Empty(t, got.RecipientPhone)
		assert.Empty(t, got.Street)
		assert.Nil(t, got.Latitude)
		assert.NotNil(t, got.AnonymizedAt)
	}
	_, err = svc.Track(ctx, abandoned.TrackingCode)
	assert.ErrorIs(t, err, apperr.ErrNotFound, "the public link of an anonymized delivery is gone")
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
	for d, want := range map[int64]int64{old.ID: 0, open.ID: 1} {
		n, err := q.CountPushSubscriptions(ctx, store.CountPushSubscriptionsParams{DeliveryID: d})
		require.NoError(t, err)
		assert.Equal(t, want, n, "only the anonymized delivery loses its browsers")
	}

	// Erased deliveries no longer change, or the recipient's data could come back.
	_, err = svc.AddEvent(ctx, owner, old.ID, delivery.EventInput{Status: delivery.StatusInTransit})
	assert.ErrorIs(t, err, apperr.ErrConflict)
	name := "Outra Pessoa"
	_, err = svc.Update(ctx, owner, old.ID, delivery.UpdateInput{RecipientName: &name})
	assert.ErrorIs(t, err, apperr.ErrConflict)
	// Not even an edit or event that read the delivery just before the job ran.
	_, err = q.UpdateDelivery(ctx, store.UpdateDeliveryParams{ID: old.ID, RecipientName: "Maria Souza", RecipientEmail: "maria@example.com"})
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	_, err = q.SetDeliveryStatus(ctx, store.SetDeliveryStatusParams{ID: old.ID, FromStatus: delivery.StatusFailed, Status: delivery.StatusInTransit})
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	n, err = job.Once(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "runs again without touching it")
}

func TestPG_DeletesExpiredKeysAndTokens(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	q := store.New(pool)
	u, err := q.CreateUser(ctx, store.CreateUserParams{CarrierID: testdb.Carrier(t, pool), Name: "Dona", Email: "dona@example.com", PasswordHash: "x", Role: auth.RoleCarrier})
	require.NoError(t, err)

	for _, key := range []string{"old", "new"} {
		_, err := q.ReserveIdempotencyKey(ctx, store.ReserveIdempotencyKeyParams{UserID: u.ID, Key: key, RequestHash: "h"})
		require.NoError(t, err)
	}
	_, err = pool.Exec(ctx, "UPDATE idempotency_keys SET created_at = now() - interval '25 hours' WHERE key = 'old'")
	require.NoError(t, err)
	for i, expires := range []time.Time{time.Now().Add(-time.Minute), time.Now().Add(time.Hour)} {
		err := q.CreateRefreshToken(ctx, store.CreateRefreshTokenParams{UserID: u.ID, FamilyID: "f", TokenHash: []byte{byte(i)}, ExpiresAt: expires})
		require.NoError(t, err)
	}

	_, err = retention.NewJob(q, 90*24*time.Hour).Once(ctx)
	require.NoError(t, err)

	var keys []string
	rows, err := pool.Query(ctx, "SELECT key FROM idempotency_keys")
	require.NoError(t, err)
	for rows.Next() {
		var k string
		require.NoError(t, rows.Scan(&k))
		keys = append(keys, k)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"new"}, keys, "a key still inside its 24 hours stays")
	var valid, all int
	err = pool.QueryRow(ctx, "SELECT count(*) FILTER (WHERE expires_at > now()), count(*) FROM refresh_tokens").Scan(&valid, &all)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 1}, []int{valid, all}, "only the expired token is gone")
}

func ptrTo[T any](v T) *T { return &v }
