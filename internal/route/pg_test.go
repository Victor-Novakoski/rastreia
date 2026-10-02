package route_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/route"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/testdb"
)

type fixture struct {
	deliveries *delivery.Service
	routes     *route.Service
	announced  *announcer
	owner      auth.Claims
	ana, bruno auth.Claims
	rival      auth.Claims
}

type announcer struct{ ids []int64 }

func (a *announcer) Announce(_ context.Context, d delivery.Delivery) { a.ids = append(a.ids, d.ID) }

func setup(t *testing.T) fixture {
	pool := testdb.New(t)
	q := store.New(pool)
	ctx := context.Background()
	user := func(carrierID int64, name, role string) auth.Claims {
		u, err := q.CreateUser(ctx, store.CreateUserParams{CarrierID: carrierID, Name: name, Email: name + "@example.com", PasswordHash: "x", Role: role})
		require.NoError(t, err)
		return auth.Claims{UserID: u.ID, Role: role, CarrierID: carrierID}
	}
	carrierID, rivalID := testdb.Carrier(t, pool), testdb.Carrier(t, pool)
	deliveries := delivery.NewService(delivery.NewPGStore(pool))
	a := &announcer{}
	return fixture{
		deliveries: deliveries,
		routes:     route.NewService(route.NewPGStore(pool), a),
		announced:  a,
		owner:      user(carrierID, "dona", auth.RoleCarrier),
		ana:        user(carrierID, "ana", auth.RoleDriver),
		bruno:      user(carrierID, "bruno", auth.RoleDriver),
		rival:      user(rivalID, "rival", auth.RoleDriver),
	}
}

// create adds a delivery at number, with a map position east of Praça da Sé
// that grows with lng.
func (f fixture) create(t *testing.T, number string, lng float64, driver *auth.Claims) delivery.Delivery {
	lat, lon := -23.55, -46.63+lng
	in := delivery.CreateInput{
		RecipientName: "Maria Souza", RecipientEmail: "maria@example.com", RecipientPhone: "11987654321",
		PostalCode: "01001000", Street: "Praça da Sé", Number: number, District: "Sé", City: "São Paulo", State: "SP",
		Latitude: &lat, Longitude: &lon,
	}
	if driver != nil {
		in.DriverID = &driver.UserID
	}
	d, err := f.deliveries.Create(context.Background(), f.owner, in)
	require.NoError(t, err)
	return d
}

func ids(r route.Route) [][]int64 {
	out := [][]int64{}
	for _, st := range r.Stops {
		var stop []int64
		for _, p := range st.Packages {
			stop = append(stop, p.ID)
		}
		out = append(out, stop)
	}
	return out
}

func numbers(r route.Route) [][]int {
	out := [][]int{}
	for _, st := range r.Stops {
		var stop []int
		for _, p := range st.Packages {
			stop = append(stop, p.Number)
		}
		out = append(out, stop)
	}
	return out
}

func TestPG_ScanAndOrder(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	r, err := f.routes.Today(ctx, f.ana)
	require.NoError(t, err)
	assert.Empty(t, r.Stops)
	assert.Equal(t, time.Now().In(time.FixedZone("BRT", -3*3600)).Format(time.DateOnly), r.Date)

	far := f.create(t, "300", 0.03, &f.ana)
	near := f.create(t, "100", 0.01, nil)
	nearAgain := f.create(t, "100", 0.01, nil)
	middle := f.create(t, "200", 0.02, &f.ana)

	for _, d := range []delivery.Delivery{far, near, nearAgain} {
		_, err = f.routes.Add(ctx, f.ana, d.TrackingCode)
		require.NoError(t, err)
	}
	r, err = f.routes.Add(ctx, f.ana, "https://rastreia.app/rastreio/"+middle.TrackingCode)
	require.NoError(t, err, "the label's QR code holds the tracking link")
	_, err = f.routes.Add(ctx, f.ana, middle.TrackingCode)
	require.NoError(t, err, "scanning twice changes nothing")

	assert.Equal(t, [][]int64{{far.ID}, {near.ID, nearAgain.ID}, {middle.ID}}, ids(r), "same address, same stop")
	assert.Equal(t, 4, r.TotalPackages)
	assert.Equal(t, [][]int{{1}, {2, 3}, {4}}, numbers(r), "packages are numbered 1..N in stop order")
	assert.Equal(t, []int64{near.ID, nearAgain.ID}, f.announced.ids, "packages without a driver were taken and the panel told")

	got, err := f.deliveries.Get(ctx, f.owner, near.ID)
	require.NoError(t, err)
	assert.Equal(t, &f.ana.UserID, got.DriverID)

	start := &route.Point{Lat: -23.55, Lng: -46.63}
	r, err = f.routes.Optimize(ctx, f.ana, start)
	require.NoError(t, err)
	assert.Equal(t, [][]int64{{near.ID, nearAgain.ID}, {middle.ID}, {far.ID}}, ids(r), "nearest first from the driver")

	r, err = f.routes.Reorder(ctx, f.ana, []int64{far.ID, middle.ID, nearAgain.ID, near.ID})
	require.NoError(t, err)
	assert.Equal(t, [][]int64{{far.ID}, {middle.ID}, {nearAgain.ID, near.ID}}, ids(r), "the driver's order wins")
	assert.Equal(t, [][]int{{1}, {2}, {3, 4}}, numbers(r))

	_, err = f.routes.Reorder(ctx, f.ana, []int64{far.ID, middle.ID, near.ID})
	var verr *apperr.ValidationError
	assert.ErrorAs(t, err, &verr, "a package left out")
	_, err = f.routes.Reorder(ctx, f.ana, []int64{far.ID, middle.ID, near.ID, near.ID})
	assert.ErrorAs(t, err, &verr, "a package twice")

	r, err = f.routes.Remove(ctx, f.ana, middle.ID)
	require.NoError(t, err)
	assert.Equal(t, [][]int64{{far.ID}, {nearAgain.ID, near.ID}}, ids(r))
	_, err = f.routes.Remove(ctx, f.ana, middle.ID)
	assert.ErrorIs(t, err, apperr.ErrNotFound)

	other, err := f.routes.Today(ctx, f.bruno)
	require.NoError(t, err)
	assert.Empty(t, other.Stops, "each driver has their own route")
}

func TestPG_ScanRefused(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	brunos := f.create(t, "10", 0, &f.bruno)
	_, err := f.routes.Add(ctx, f.ana, brunos.TrackingCode)
	assert.ErrorIs(t, err, apperr.ErrConflict, "another driver's package")

	_, err = f.routes.Add(ctx, f.rival, brunos.TrackingCode)
	assert.ErrorIs(t, err, apperr.ErrNotFound, "another carrier's package does not exist")
	_, err = f.routes.Add(ctx, f.ana, "RSNOPE")
	assert.ErrorIs(t, err, apperr.ErrNotFound)

	done := f.create(t, "20", 0, &f.ana)
	for _, st := range []string{delivery.StatusPickedUp, delivery.StatusInTransit, delivery.StatusDelivered} {
		_, err = f.deliveries.AddEvent(ctx, f.ana, done.ID, delivery.EventInput{Status: st})
		require.NoError(t, err)
	}
	_, err = f.routes.Add(ctx, f.ana, done.TrackingCode)
	assert.ErrorIs(t, err, apperr.ErrConflict, "already delivered")

	r, err := f.routes.Today(ctx, f.ana)
	require.NoError(t, err)
	assert.Empty(t, r.Stops)
}
