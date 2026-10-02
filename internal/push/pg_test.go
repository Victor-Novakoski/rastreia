package push_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/push"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/testdb"
)

func TestPG_SubscribeListAndForget(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	q := store.New(pool)
	carrierID := testdb.Carrier(t, pool)
	u, err := q.CreateUser(ctx, store.CreateUserParams{CarrierID: carrierID, Name: "Dona", Email: "dona@example.com", PasswordHash: "x", Role: auth.RoleCarrier})
	require.NoError(t, err)
	deliveries := delivery.NewService(delivery.NewPGStore(pool))
	owner := auth.Claims{UserID: u.ID, Role: auth.RoleCarrier, CarrierID: carrierID}
	d, err := deliveries.Create(ctx, owner, delivery.CreateInput{
		RecipientName: "Maria Souza", RecipientEmail: "maria@example.com", Address: "Rua A, 10",
	})
	require.NoError(t, err)
	svc := push.NewService(deliveries, q)

	s := sub("https://fcm.googleapis.com/fcm/send/abc")
	require.NoError(t, svc.Subscribe(ctx, d.TrackingCode, s))
	require.NoError(t, svc.Subscribe(ctx, d.TrackingCode, s), "same browser again is an update")
	require.NoError(t, svc.Subscribe(ctx, d.TrackingCode, sub("https://web.push.apple.com/xyz")))

	list, err := q.ListPushSubscriptionsByCode(ctx, d.TrackingCode)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	require.NoError(t, svc.Unsubscribe(ctx, d.TrackingCode, "https://web.push.apple.com/xyz"))
	list, err = q.ListPushSubscriptionsByCode(ctx, d.TrackingCode)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, q.DeletePushSubscriptionsByCode(ctx, d.TrackingCode))
	list, err = q.ListPushSubscriptionsByCode(ctx, d.TrackingCode)
	require.NoError(t, err)
	assert.Empty(t, list)
}
