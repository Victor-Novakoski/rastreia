package notify_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/notify"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/testdb"
)

type fakeSender struct {
	msgs   []notify.StatusChanged
	failAt int // 1-based call that fails; 0 never fails
}

func (f *fakeSender) Publish(_ context.Context, _ string, body []byte) error {
	if f.failAt == len(f.msgs)+1 {
		return errors.New("rabbitmq down")
	}
	var m notify.StatusChanged
	if err := json.Unmarshal(body, &m); err != nil {
		return err
	}
	f.msgs = append(f.msgs, m)
	return nil
}

// newDelivery creates a delivery and moves it to picked_up: two events.
func newDelivery(t *testing.T, pool *pgxpool.Pool) delivery.Delivery {
	t.Helper()
	ctx := context.Background()
	carrierID := testdb.Carrier(t, pool)
	u, err := store.New(pool).CreateUser(ctx, store.CreateUserParams{
		CarrierID: carrierID, Name: "Dona", Email: "dona@example.com", PasswordHash: "x", Role: auth.RoleCarrier,
	})
	require.NoError(t, err)
	owner := auth.Claims{UserID: u.ID, Role: auth.RoleCarrier, CarrierID: carrierID}
	svc := delivery.NewService(delivery.NewPGStore(pool))
	d, err := svc.Create(ctx, owner, delivery.CreateInput{
		RecipientName: "Maria Souza", RecipientEmail: "maria@example.com", Address: "Rua A, 10",
	})
	require.NoError(t, err)
	_, err = svc.AddEvent(ctx, owner, d.ID,
		delivery.EventInput{Status: delivery.StatusPickedUp})
	require.NoError(t, err)
	return d
}

func unpublished(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT count(*) FROM delivery_events WHERE published_at IS NULL").Scan(&n))
	return n
}

func TestRelay_PublishesEachEventOnce(t *testing.T) {
	pool := testdb.New(t)
	d := newDelivery(t, pool)
	sender := &fakeSender{}
	relay := notify.NewRelay(pool, sender)

	n, err := relay.Flush(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	require.Len(t, sender.msgs, 2)
	assert.Equal(t, "pending", sender.msgs[0].Status)
	assert.Equal(t, "picked_up", sender.msgs[1].Status)
	assert.Equal(t, d.TrackingCode, sender.msgs[1].TrackingCode)
	assert.Equal(t, "maria@example.com", sender.msgs[1].RecipientEmail)
	assert.Equal(t, 0, unpublished(t, pool))

	n, err = relay.Flush(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n, "already published")
}

func TestRelay_KeepsEventsWhenPublishFails(t *testing.T) {
	pool := testdb.New(t)
	newDelivery(t, pool)
	sender := &fakeSender{failAt: 2}
	relay := notify.NewRelay(pool, sender)

	n, err := relay.Flush(context.Background())
	assert.Error(t, err)
	assert.Equal(t, 1, n, "the first one went out and is marked")
	assert.Equal(t, 1, unpublished(t, pool))

	sender.failAt = 0
	_, err = relay.Flush(context.Background())
	require.NoError(t, err)
	require.Len(t, sender.msgs, 2)
	assert.Equal(t, "picked_up", sender.msgs[1].Status)
}

func TestRelay_SkipsStaleEvents(t *testing.T) {
	pool := testdb.New(t)
	newDelivery(t, pool)
	_, err := pool.Exec(context.Background(),
		"UPDATE delivery_events SET created_at = now() - interval '2 hours' WHERE status = 'pending'")
	require.NoError(t, err)
	sender := &fakeSender{}
	relay := notify.NewRelay(pool, sender)
	relay.MaxAge = time.Hour

	_, err = relay.Flush(context.Background())
	require.NoError(t, err)
	require.Len(t, sender.msgs, 1)
	assert.Equal(t, "picked_up", sender.msgs[0].Status)
	assert.Equal(t, 0, unpublished(t, pool))
}
