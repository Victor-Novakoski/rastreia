package delivery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
)

var (
	admin   = auth.Claims{UserID: 1, Role: auth.RoleAdmin}
	driverA = auth.Claims{UserID: 2, Role: auth.RoleDriver}
	driverB = auth.Claims{UserID: 3, Role: auth.RoleDriver}
)

// newAssigned creates a delivery assigned to driverA.
func newAssigned(t *testing.T, svc *Service) Delivery {
	t.Helper()
	in := validInput()
	in.DriverID = ptr(driverA.UserID)
	d, err := svc.Create(context.Background(), admin.UserID, in)
	require.NoError(t, err)
	return d
}

func TestAddEvent_FullJourney(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)
	d := newAssigned(t, svc)
	ctx := context.Background()

	steps := []EventInput{
		{Status: StatusPickedUp},
		{Status: StatusInTransit},
		{Status: StatusFailed, Note: ptr(" Destinatário ausente ")},
		{Status: StatusInTransit},
		{Status: StatusDelivered},
	}
	for _, in := range steps {
		ev, err := svc.AddEvent(ctx, driverA, d.ID, in)
		require.NoError(t, err, in.Status)
		assert.Equal(t, in.Status, ev.Status)
		assert.Equal(t, ptr(driverA.UserID), ev.CreatedBy)

		got := fs.deliveries[d.ID]
		assert.Equal(t, in.Status, got.Status)
		if in.Status == StatusFailed || in.Status == StatusDelivered {
			assert.NotNil(t, got.CompletedAt, "%s sets completed_at", in.Status)
		} else {
			assert.Nil(t, got.CompletedAt, "%s clears completed_at", in.Status)
		}
	}

	events, err := svc.ListEvents(ctx, driverA, d.ID)
	require.NoError(t, err)
	require.Len(t, events, 6, "pending + 5 steps")
	assert.Equal(t, ptr("Destinatário ausente"), events[3].Note, "note is trimmed")
}

func TestAddEvent_InvalidTransitions(t *testing.T) {
	cases := map[string]struct {
		path []string
		next string
	}{
		"skip to delivered":     {nil, StatusDelivered},
		"same status":           {nil, StatusPending},
		"back to pending":       {[]string{StatusPickedUp}, StatusPending},
		"delivered is final":    {[]string{StatusPickedUp, StatusInTransit, StatusDelivered}, StatusInTransit},
		"failed cannot deliver": {[]string{StatusFailed}, StatusDelivered},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc := NewService(newFakeStore())
			d := newAssigned(t, svc)
			for _, st := range tc.path {
				_, err := svc.AddEvent(context.Background(), admin, d.ID, EventInput{Status: st, Note: ptr("x")})
				require.NoError(t, err)
			}
			_, err := svc.AddEvent(context.Background(), admin, d.ID, EventInput{Status: tc.next, Note: ptr("x")})
			assert.ErrorIs(t, err, apperr.ErrConflict)
		})
	}
}

func TestAddEvent_Validation(t *testing.T) {
	svc := NewService(newFakeStore())
	d := newAssigned(t, svc)

	cases := map[string]struct {
		in    EventInput
		field string
	}{
		"unknown status":      {EventInput{Status: "lost"}, "status"},
		"failed without note": {EventInput{Status: StatusFailed, Note: ptr("   ")}, "note"},
		"note over 500 chars": {EventInput{Status: StatusPickedUp, Note: ptr(string(make([]rune, 501)))}, "note"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.AddEvent(context.Background(), admin, d.ID, tc.in)
			var verr *apperr.ValidationError
			require.ErrorAs(t, err, &verr)
			assert.Contains(t, verr.Fields, tc.field)
		})
	}
}

func TestEvents_DriverOnlySeesOwnDeliveries(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)
	ctx := context.Background()
	mine := newAssigned(t, svc)
	unassigned, err := svc.Create(ctx, admin.UserID, validInput())
	require.NoError(t, err)

	for _, id := range []int64{mine.ID, unassigned.ID, 999} {
		_, err := svc.AddEvent(ctx, driverB, id, EventInput{Status: StatusPickedUp})
		assert.ErrorIs(t, err, apperr.ErrNotFound, "driver B must not change delivery %d", id)
		_, err = svc.ListEvents(ctx, driverB, id)
		assert.ErrorIs(t, err, apperr.ErrNotFound, "driver B must not read delivery %d", id)
	}
	assert.Equal(t, StatusPending, fs.deliveries[mine.ID].Status)

	_, err = svc.AddEvent(ctx, driverA, unassigned.ID, EventInput{Status: StatusPickedUp})
	assert.ErrorIs(t, err, apperr.ErrNotFound, "unassigned deliveries are admin only")

	_, err = svc.AddEvent(ctx, admin, unassigned.ID, EventInput{Status: StatusPickedUp})
	assert.NoError(t, err, "admins reach every delivery")
}

func TestListForDriver_FiltersByCaller(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)
	newAssigned(t, svc)

	list, err := svc.ListForDriver(context.Background(), driverB.UserID, ListInput{Page: 2, Size: 5})
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.Equal(t, driverB.UserID, fs.lastMine.DriverID)
	assert.Equal(t, int32(5), fs.lastMine.Offset)

	list, err = svc.ListForDriver(context.Background(), driverA.UserID, ListInput{})
	require.NoError(t, err)
	assert.Len(t, list, 1)

	_, err = svc.ListForDriver(context.Background(), driverA.UserID, ListInput{Status: ptr("lost")})
	var verr *apperr.ValidationError
	assert.ErrorAs(t, err, &verr)
}

func TestTrack_HidesPersonalData(t *testing.T) {
	svc := NewService(newFakeStore())
	d := newAssigned(t, svc)
	_, err := svc.AddEvent(context.Background(), driverA, d.ID, EventInput{Status: StatusPickedUp, Note: ptr("deixei com o vizinho do 32")})
	require.NoError(t, err)

	tr, err := svc.Track(context.Background(), " "+strings.ToLower(d.TrackingCode)+" ")
	require.NoError(t, err, "codes are case-insensitive and trimmed")
	assert.Equal(t, "Maria", tr.RecipientFirstName)
	assert.Equal(t, StatusPickedUp, tr.Status)
	require.Len(t, tr.Events, 2)

	body, err := json.Marshal(tr)
	require.NoError(t, err)
	for _, secret := range []string{"Souza", "maria@example.com", "Rua A", "vizinho", "driver", `"id"`} {
		assert.NotContains(t, string(body), secret)
	}
}

func TestTrack_NotFound(t *testing.T) {
	svc := NewService(newFakeStore())
	for _, code := range []string{"", "RS123", "RSAAAAAAAAAA", "RS0OOOOOOOOO", "1 OR 1=1"} {
		_, err := svc.Track(context.Background(), code)
		assert.ErrorIs(t, err, apperr.ErrNotFound, code)
	}
}

func TestTrack_ExpiresThirtyDaysAfterCompletion(t *testing.T) {
	svc := NewService(newFakeStore())
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return start }
	d := newAssigned(t, svc)
	for _, st := range []string{StatusPickedUp, StatusInTransit} {
		_, err := svc.AddEvent(context.Background(), admin, d.ID, EventInput{Status: st})
		require.NoError(t, err)
	}

	svc.now = func() time.Time { return start.AddDate(0, 6, 0) }
	_, err := svc.Track(context.Background(), d.TrackingCode)
	assert.NoError(t, err, "an open delivery never expires")

	svc.now = func() time.Time { return start }
	_, err = svc.AddEvent(context.Background(), admin, d.ID, EventInput{Status: StatusDelivered})
	require.NoError(t, err)

	svc.now = func() time.Time { return start.Add(trackingTTL - time.Minute) }
	_, err = svc.Track(context.Background(), d.TrackingCode)
	assert.NoError(t, err)

	svc.now = func() time.Time { return start.Add(trackingTTL + time.Minute) }
	_, err = svc.Track(context.Background(), d.TrackingCode)
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
