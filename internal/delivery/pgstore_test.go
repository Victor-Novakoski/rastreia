package delivery_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/testdb"
)

// setup returns a service on a fresh database with one owner and one driver.
func setup(t *testing.T) (*delivery.Service, *pgxpool.Pool, auth.Claims, auth.Claims) {
	t.Helper()
	pool := testdb.New(t)
	q := store.New(pool)
	ctx := context.Background()
	carrierID := testdb.Carrier(t, pool)
	a, err := q.CreateUser(ctx, store.CreateUserParams{CarrierID: carrierID, Name: "Dona", Email: "dona@example.com", PasswordHash: "x", Role: auth.RoleCarrier})
	require.NoError(t, err)
	d, err := q.CreateUser(ctx, store.CreateUserParams{CarrierID: carrierID, Name: "Ana", Email: "ana@example.com", PasswordHash: "x", Role: auth.RoleDriver})
	require.NoError(t, err)
	return delivery.NewService(delivery.NewPGStore(pool)),
		pool,
		auth.Claims{UserID: a.ID, Role: auth.RoleCarrier, CarrierID: carrierID},
		auth.Claims{UserID: d.ID, Role: auth.RoleDriver, CarrierID: carrierID}
}

func input(driverID int64) delivery.CreateInput {
	in := address()
	in.DriverID = &driverID
	return in
}

func address() delivery.CreateInput {
	return delivery.CreateInput{
		RecipientName: "Maria Souza", RecipientEmail: "maria@example.com", RecipientPhone: "11987654321",
		PostalCode: "01001000", Street: "Praça da Sé", Number: "10", District: "Sé", City: "São Paulo", State: "SP",
	}
}

func count(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n))
	return n
}

func TestPG_JourneyAndTracking(t *testing.T) {
	svc, _, owner, driver := setup(t)
	ctx := context.Background()

	d, err := svc.Create(ctx, owner, input(driver.UserID))
	require.NoError(t, err)
	for _, st := range []string{delivery.StatusPickedUp, delivery.StatusInTransit, delivery.StatusDelivered} {
		_, err := svc.AddEvent(ctx, driver, d.ID, delivery.EventInput{Status: st})
		require.NoError(t, err, st)
	}

	got, err := svc.Get(ctx, owner, d.ID)
	require.NoError(t, err)
	assert.Equal(t, delivery.StatusDelivered, got.Status)
	assert.NotNil(t, got.CompletedAt)

	tr, err := svc.Track(ctx, d.TrackingCode)
	require.NoError(t, err)
	assert.Equal(t, "Maria", tr.RecipientFirstName)
	assert.Equal(t, "Transportadora Teste", tr.CarrierName)
	require.Len(t, tr.Events, 4)
	assert.Equal(t, delivery.StatusPending, tr.Events[0].Status)

	mine, err := svc.ListForDriver(ctx, driver.UserID, delivery.ListInput{Status: ptr(delivery.StatusDelivered)})
	require.NoError(t, err)
	assert.Len(t, mine, 1)
}

func TestPG_ConcurrentEventsApplyOnce(t *testing.T) {
	svc, pool, owner, driver := setup(t)
	ctx := context.Background()
	d, err := svc.Create(ctx, owner, input(driver.UserID))
	require.NoError(t, err)

	const n = 10
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			_, errs[i] = svc.AddEvent(ctx, driver, d.ID, delivery.EventInput{Status: delivery.StatusPickedUp})
		})
	}
	wg.Wait()

	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else {
			assert.ErrorIs(t, err, apperr.ErrConflict)
		}
	}
	assert.Equal(t, 1, ok, "exactly one of the simultaneous events wins")
	assert.Equal(t, 2, count(t, pool, "delivery_events"), "pending + picked_up, no duplicates")
}

func TestPG_IdempotentCreateUnderConcurrency(t *testing.T) {
	svc, pool, owner, driver := setup(t)
	ctx := context.Background()

	const n = 10
	ids := make([]int64, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			d, _, err := svc.CreateIdempotent(ctx, owner, "retry-123", input(driver.UserID))
			ids[i], errs[i] = d.ID, err
		})
	}
	wg.Wait()

	for i := range n {
		require.NoError(t, errs[i])
		assert.Equal(t, ids[0], ids[i])
	}
	assert.Equal(t, 1, count(t, pool, "deliveries"))
	assert.Equal(t, 1, count(t, pool, "delivery_events"))
}

func TestPG_FailedTransactionLeavesNoTrace(t *testing.T) {
	svc, pool, owner, _ := setup(t)
	// The first event references a user that does not exist, so its insert
	// fails after the delivery row was written.
	ghost := owner
	ghost.UserID = 999_999
	_, err := svc.Create(context.Background(), ghost, address())
	require.Error(t, err)
	assert.Equal(t, 0, count(t, pool, "deliveries"), "the delivery is rolled back with the event")
}

func ptr[T any](v T) *T { return &v }

func TestPG_SearchByCodeNameOrEmail(t *testing.T) {
	svc, pool, owner, driver := setup(t)
	ctx := context.Background()
	create := func(name, email string) delivery.Delivery {
		in := input(driver.UserID)
		in.RecipientName, in.RecipientEmail = name, email
		d, err := svc.Create(ctx, owner, in)
		require.NoError(t, err)
		return d
	}
	joao := create("João da Conceição", "joao@example.com")
	maria := create("Maria Souza", "maria.souza@exemplo.com.br")
	percent := create("Loja 50% Off", "loja@example.com")

	// Another carrier's delivery never shows up, whatever the search.
	other := owner
	other.CarrierID = testdb.Carrier(t, pool)
	_, err := svc.Create(ctx, other, address())
	require.NoError(t, err)

	cases := []struct {
		q    string
		want []int64
	}{
		{"joao", []int64{joao.ID}},
		{"JOÃO DA", []int64{joao.ID}},
		{"conceicao", []int64{joao.ID}},
		{"souza", []int64{maria.ID}},
		{"exemplo.com", []int64{maria.ID}},
		{strings.ToLower(maria.TrackingCode[2:8]), []int64{maria.ID}},
		{maria.TrackingCode, []int64{maria.ID}},
		{"50%", []int64{percent.ID}},
		{"%", []int64{percent.ID}},
		{"_", nil},
		{"example.com", []int64{percent.ID, joao.ID}},
		{"ninguém", nil},
	}
	for _, tc := range cases {
		list, err := svc.List(ctx, owner.CarrierID, delivery.ListInput{Search: tc.q})
		require.NoError(t, err, tc.q)
		var ids []int64
		for _, d := range list {
			ids = append(ids, d.ID)
		}
		assert.Equal(t, tc.want, ids, "search %q", tc.q)
	}

	list, err := svc.List(ctx, owner.CarrierID, delivery.ListInput{Search: "souza", Status: ptr(delivery.StatusDelivered)})
	require.NoError(t, err)
	assert.Empty(t, list, "the search and the status filter add up")
}
