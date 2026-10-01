package delivery

import (
	"context"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// fakeStore keeps deliveries in memory. createErrs are returned, in order,
// by the next CreateDelivery calls. InTx undoes the changes when fn fails,
// like a rollback.
type fakeStore struct {
	deliveries map[int64]store.Delivery
	events     []store.DeliveryEvent
	keys       map[store.GetIdempotencyKeyParams]store.IdempotencyKey
	users      map[int64]store.User
	createErrs []error
	lastList   store.ListDeliveriesParams
	lastMine   store.ListDriverDeliveriesParams
	nextID     int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		deliveries: map[int64]store.Delivery{},
		keys:       map[store.GetIdempotencyKeyParams]store.IdempotencyKey{},
		users: map[int64]store.User{
			1: {ID: 1, Role: auth.RoleAdmin},
			2: {ID: 2, Role: auth.RoleDriver},
			3: {ID: 3, Role: auth.RoleDriver},
		},
	}
}

func (f *fakeStore) InTx(_ context.Context, fn func(Store) error) error {
	deliveries, events, keys := maps.Clone(f.deliveries), slices.Clone(f.events), maps.Clone(f.keys)
	err := fn(f)
	if err != nil {
		f.deliveries, f.events, f.keys = deliveries, events, keys
	}
	return err
}

func (f *fakeStore) GetDeliveryByTrackingCode(_ context.Context, code string) (store.Delivery, error) {
	for _, d := range f.deliveries {
		if d.TrackingCode == code {
			return d, nil
		}
	}
	return store.Delivery{}, pgx.ErrNoRows
}

func (f *fakeStore) ListDriverDeliveries(_ context.Context, arg store.ListDriverDeliveriesParams) ([]store.Delivery, error) {
	f.lastMine = arg
	var out []store.Delivery
	for _, d := range f.deliveries {
		if d.DriverID != nil && *d.DriverID == arg.DriverID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeStore) SetDeliveryStatus(_ context.Context, arg store.SetDeliveryStatusParams) (store.Delivery, error) {
	d, ok := f.deliveries[arg.ID]
	if !ok || d.Status != arg.FromStatus {
		return store.Delivery{}, pgx.ErrNoRows
	}
	d.Status, d.CompletedAt = arg.Status, arg.CompletedAt
	f.deliveries[d.ID] = d
	return d, nil
}

func (f *fakeStore) CreateDeliveryEvent(_ context.Context, arg store.CreateDeliveryEventParams) (store.DeliveryEvent, error) {
	e := store.DeliveryEvent{
		ID: int64(len(f.events) + 1), DeliveryID: arg.DeliveryID, Status: arg.Status,
		Note: arg.Note, CreatedBy: arg.CreatedBy, CreatedAt: time.Now(),
	}
	f.events = append(f.events, e)
	return e, nil
}

func (f *fakeStore) ListDeliveryEvents(_ context.Context, deliveryID int64) ([]store.DeliveryEvent, error) {
	var out []store.DeliveryEvent
	for _, e := range f.events {
		if e.DeliveryID == deliveryID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeStore) DeleteExpiredIdempotencyKey(context.Context, store.DeleteExpiredIdempotencyKeyParams) error {
	return nil
}

func (f *fakeStore) ReserveIdempotencyKey(_ context.Context, arg store.ReserveIdempotencyKeyParams) (store.IdempotencyKey, error) {
	k := store.GetIdempotencyKeyParams{UserID: arg.UserID, Key: arg.Key}
	if _, exists := f.keys[k]; exists {
		return store.IdempotencyKey{}, pgx.ErrNoRows
	}
	f.keys[k] = store.IdempotencyKey{UserID: arg.UserID, Key: arg.Key, RequestHash: arg.RequestHash}
	return f.keys[k], nil
}

func (f *fakeStore) GetIdempotencyKey(_ context.Context, arg store.GetIdempotencyKeyParams) (store.IdempotencyKey, error) {
	k, ok := f.keys[arg]
	if !ok {
		return store.IdempotencyKey{}, pgx.ErrNoRows
	}
	return k, nil
}

func (f *fakeStore) SetIdempotencyKeyDelivery(_ context.Context, arg store.SetIdempotencyKeyDeliveryParams) error {
	k := store.GetIdempotencyKeyParams{UserID: arg.UserID, Key: arg.Key}
	saved := f.keys[k]
	saved.DeliveryID = arg.DeliveryID
	f.keys[k] = saved
	return nil
}

func (f *fakeStore) CreateDelivery(_ context.Context, arg store.CreateDeliveryParams) (store.Delivery, error) {
	if len(f.createErrs) > 0 {
		err := f.createErrs[0]
		f.createErrs = f.createErrs[1:]
		return store.Delivery{}, err
	}
	f.nextID++
	d := store.Delivery{
		ID: f.nextID, TrackingCode: arg.TrackingCode, RecipientName: arg.RecipientName,
		RecipientEmail: arg.RecipientEmail, Address: arg.Address, DriverID: arg.DriverID, Status: StatusPending,
	}
	f.deliveries[d.ID] = d
	return d, nil
}

func (f *fakeStore) GetDelivery(_ context.Context, id int64) (store.Delivery, error) {
	d, ok := f.deliveries[id]
	if !ok {
		return store.Delivery{}, pgx.ErrNoRows
	}
	return d, nil
}

func (f *fakeStore) ListDeliveries(_ context.Context, arg store.ListDeliveriesParams) ([]store.Delivery, error) {
	f.lastList = arg
	var out []store.Delivery
	for _, d := range f.deliveries {
		out = append(out, d)
	}
	return out, nil
}

func (f *fakeStore) UpdateDelivery(_ context.Context, arg store.UpdateDeliveryParams) (store.Delivery, error) {
	d, ok := f.deliveries[arg.ID]
	if !ok {
		return store.Delivery{}, pgx.ErrNoRows
	}
	if arg.RecipientName != nil {
		d.RecipientName = *arg.RecipientName
	}
	if arg.RecipientEmail != nil {
		d.RecipientEmail = *arg.RecipientEmail
	}
	if arg.Address != nil {
		d.Address = *arg.Address
	}
	if arg.DriverID != nil {
		d.DriverID = arg.DriverID
	}
	f.deliveries[d.ID] = d
	return d, nil
}

func (f *fakeStore) GetUserByID(_ context.Context, id int64) (store.User, error) {
	u, ok := f.users[id]
	if !ok {
		return store.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func ptr[T any](v T) *T { return &v }

const adminID = 1

func validInput() CreateInput {
	return CreateInput{RecipientName: " Maria Souza ", RecipientEmail: "Maria@Example.com", Address: "Rua A, 10"}
}

func TestCreate(t *testing.T) {
	svc := NewService(newFakeStore())

	d, err := svc.Create(context.Background(), adminID, validInput())
	require.NoError(t, err)
	assert.Equal(t, "Maria Souza", d.RecipientName)
	assert.Equal(t, "maria@example.com", d.RecipientEmail)
	assert.Equal(t, StatusPending, d.Status)
	assert.Regexp(t, regexp.MustCompile(`^RS[A-Z2-9]{10}$`), d.TrackingCode)

	events, err := svc.ListEvents(context.Background(), admin, d.ID)
	require.NoError(t, err)
	require.Len(t, events, 1, "creating a delivery records the first event")
	assert.Equal(t, StatusPending, events[0].Status)
	assert.Equal(t, ptr(int64(adminID)), events[0].CreatedBy)
}

func TestCreate_Validation(t *testing.T) {
	svc := NewService(newFakeStore())

	cases := map[string]struct {
		mutate func(*CreateInput)
		field  string
	}{
		"empty name":      {func(in *CreateInput) { in.RecipientName = "  " }, "recipient_name"},
		"bad e-mail":      {func(in *CreateInput) { in.RecipientEmail = "maria" }, "recipient_email"},
		"empty address":   {func(in *CreateInput) { in.Address = "" }, "address"},
		"unknown driver":  {func(in *CreateInput) { in.DriverID = ptr(int64(99)) }, "driver_id"},
		"admin as driver": {func(in *CreateInput) { in.DriverID = ptr(int64(1)) }, "driver_id"},
		"long name":       {func(in *CreateInput) { in.RecipientName = strings.Repeat("a", 121) }, "recipient_name"},
		"long address":    {func(in *CreateInput) { in.Address = strings.Repeat("a", 301) }, "address"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)
			_, err := svc.Create(context.Background(), adminID, in)

			var verr *apperr.ValidationError
			require.ErrorAs(t, err, &verr)
			assert.Contains(t, verr.Fields, tc.field)
		})
	}
}

func TestCreate_WithDriver(t *testing.T) {
	svc := NewService(newFakeStore())
	in := validInput()
	in.DriverID = ptr(int64(2))

	d, err := svc.Create(context.Background(), adminID, in)
	require.NoError(t, err)
	assert.Equal(t, ptr(int64(2)), d.DriverID)
}

func TestCreate_RetriesOnCodeCollision(t *testing.T) {
	fs := newFakeStore()
	fs.createErrs = []error{&pgconn.PgError{Code: "23505"}}
	svc := NewService(fs)

	_, err := svc.Create(context.Background(), adminID, validInput())
	require.NoError(t, err)
	assert.Len(t, fs.deliveries, 1)
}

func TestGetAndUpdate_NotFound(t *testing.T) {
	svc := NewService(newFakeStore())

	_, err := svc.Get(context.Background(), 404)
	assert.ErrorIs(t, err, apperr.ErrNotFound)

	_, err = svc.Update(context.Background(), 404, UpdateInput{Address: ptr("Rua B")})
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestUpdate(t *testing.T) {
	svc := NewService(newFakeStore())
	created, err := svc.Create(context.Background(), adminID, validInput())
	require.NoError(t, err)

	updated, err := svc.Update(context.Background(), created.ID, UpdateInput{Address: ptr("  Rua B, 20 "), DriverID: ptr(int64(2))})
	require.NoError(t, err)
	assert.Equal(t, "Rua B, 20", updated.Address)
	assert.Equal(t, created.RecipientName, updated.RecipientName, "fields not sent stay the same")
	assert.Equal(t, ptr(int64(2)), updated.DriverID)

	_, err = svc.Update(context.Background(), created.ID, UpdateInput{RecipientEmail: ptr("nope")})
	var verr *apperr.ValidationError
	assert.ErrorAs(t, err, &verr)
}

func TestList_Paging(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)

	_, err := svc.List(context.Background(), ListInput{})
	require.NoError(t, err)
	assert.Equal(t, store.ListDeliveriesParams{Limit: 20, Offset: 0}, fs.lastList)

	_, err = svc.List(context.Background(), ListInput{Page: 3, Size: 10, Status: ptr(StatusDelivered)})
	require.NoError(t, err)
	assert.Equal(t, store.ListDeliveriesParams{Status: ptr(StatusDelivered), Limit: 10, Offset: 20}, fs.lastList)

	_, err = svc.List(context.Background(), ListInput{Size: 1000})
	require.NoError(t, err)
	assert.Equal(t, int32(20), fs.lastList.Limit, "oversized pages fall back to the default")

	_, err = svc.List(context.Background(), ListInput{Status: ptr("lost")})
	var verr *apperr.ValidationError
	assert.ErrorAs(t, err, &verr)

	_, err = svc.List(context.Background(), ListInput{Page: 999_999_999})
	require.ErrorAs(t, err, &verr, "a huge page would overflow the offset")
	assert.Contains(t, verr.Fields, "page")
}

func TestNewTrackingCode_IsRandom(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		code, err := NewTrackingCode()
		require.NoError(t, err)
		require.False(t, seen[code], "duplicate code %s", code)
		seen[code] = true
	}
}
