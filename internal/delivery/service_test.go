package delivery

import (
	"context"
	"errors"
	"fmt"
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
			1: {ID: 1, Role: auth.RoleCarrier, CarrierID: 1},
			2: {ID: 2, Role: auth.RoleDriver, CarrierID: 1},
			3: {ID: 3, Role: auth.RoleDriver, CarrierID: 1},
			4: {ID: 4, Role: auth.RoleCarrier, CarrierID: 2},
			5: {ID: 5, Role: auth.RoleDriver, CarrierID: 2},
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
	if !ok || d.Status != arg.FromStatus || d.AnonymizedAt != nil {
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
		RecipientEmail: arg.RecipientEmail, RecipientPhone: arg.RecipientPhone, Address: arg.Address,
		PostalCode: arg.PostalCode, Street: arg.Street, Number: arg.Number, Complement: arg.Complement,
		District: arg.District, City: arg.City, State: arg.State, AddressReference: arg.AddressReference,
		Latitude: arg.Latitude, Longitude: arg.Longitude,
		DriverID: arg.DriverID, Status: StatusPending, CarrierID: arg.CarrierID, CreatedAt: time.Now(),
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
		if d.CarrierID == arg.CarrierID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeStore) CountDeliveriesByStatus(_ context.Context, arg store.CountDeliveriesByStatusParams) ([]store.CountDeliveriesByStatusRow, error) {
	counts := map[string]int64{}
	for _, d := range f.deliveries {
		if d.CarrierID == arg.CarrierID && !d.CreatedAt.Before(arg.Since) {
			counts[d.Status]++
		}
	}
	var out []store.CountDeliveriesByStatusRow
	for st, n := range counts {
		out = append(out, store.CountDeliveriesByStatusRow{Status: st, Total: n})
	}
	return out, nil
}

func (f *fakeStore) CountUnassignedDeliveries(_ context.Context, carrierID int64) (int64, error) {
	var n int64
	for _, d := range f.deliveries {
		if d.CarrierID == carrierID && d.DriverID == nil && d.Status == StatusPending {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) UpdateDelivery(_ context.Context, arg store.UpdateDeliveryParams) (store.Delivery, error) {
	d, ok := f.deliveries[arg.ID]
	if !ok || d.AnonymizedAt != nil {
		return store.Delivery{}, pgx.ErrNoRows
	}
	d.RecipientName, d.RecipientEmail, d.RecipientPhone = arg.RecipientName, arg.RecipientEmail, arg.RecipientPhone
	d.Address, d.PostalCode, d.Street, d.Number = arg.Address, arg.PostalCode, arg.Street, arg.Number
	d.Complement, d.District, d.City, d.State = arg.Complement, arg.District, arg.City, arg.State
	d.AddressReference, d.Latitude, d.Longitude = arg.AddressReference, arg.Latitude, arg.Longitude
	if arg.DriverID != nil {
		d.DriverID = arg.DriverID
	}
	f.deliveries[d.ID] = d
	return d, nil
}

func (f *fakeStore) GetCarrier(_ context.Context, id int64) (store.Carrier, error) {
	return store.Carrier{ID: id, Name: fmt.Sprintf("Transportadora %d", id)}, nil
}

func (f *fakeStore) GetUserByID(_ context.Context, id int64) (store.User, error) {
	u, ok := f.users[id]
	if !ok {
		return store.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func ptr[T any](v T) *T { return &v }

const ownerID = 1

func validInput() CreateInput {
	return CreateInput{
		RecipientName: " Maria Souza ", RecipientEmail: "Maria@Example.com", RecipientPhone: "(11) 98765-4321",
		PostalCode: "01001-000", Street: "Praça da Sé", Number: "10", Complement: "Apto 2",
		District: "Sé", City: "São Paulo", State: "sp", AddressReference: "Portão azul",
		Latitude: ptr(-23.5503), Longitude: ptr(-46.6339),
	}
}

func TestCreate(t *testing.T) {
	svc := NewService(newFakeStore())

	d, err := svc.Create(context.Background(), owner, validInput())
	require.NoError(t, err)
	assert.Equal(t, "Maria Souza", d.RecipientName)
	assert.Equal(t, "maria@example.com", d.RecipientEmail)
	assert.Equal(t, StatusPending, d.Status)
	assert.Equal(t, "11987654321", d.RecipientPhone, "only the digits are kept")
	assert.Equal(t, "01001000", d.PostalCode)
	assert.Equal(t, "SP", d.State)
	assert.Equal(t, "Praça da Sé, 10, Apto 2 - Sé, São Paulo - SP, 01001-000", d.Address)
	assert.Equal(t, ptr(-23.5503), d.Latitude)
	assert.Regexp(t, regexp.MustCompile(`^RS[A-Z2-9]{10}$`), d.TrackingCode)

	events, err := svc.ListEvents(context.Background(), owner, d.ID)
	require.NoError(t, err)
	require.Len(t, events, 1, "creating a delivery records the first event")
	assert.Equal(t, StatusPending, events[0].Status)
	assert.Equal(t, ptr(int64(ownerID)), events[0].CreatedBy)
}

func TestCreate_Validation(t *testing.T) {
	svc := NewService(newFakeStore())

	cases := map[string]struct {
		mutate func(*CreateInput)
		field  string
	}{
		"empty name":             {func(in *CreateInput) { in.RecipientName = "  " }, "recipient_name"},
		"bad e-mail":             {func(in *CreateInput) { in.RecipientEmail = "maria" }, "recipient_email"},
		"short CEP":              {func(in *CreateInput) { in.PostalCode = "0100-100" }, "postal_code"},
		"empty street":           {func(in *CreateInput) { in.Street = " " }, "street"},
		"empty number":           {func(in *CreateInput) { in.Number = "" }, "number"},
		"empty district":         {func(in *CreateInput) { in.District = "" }, "district"},
		"empty city":             {func(in *CreateInput) { in.City = "" }, "city"},
		"unknown state":          {func(in *CreateInput) { in.State = "XX" }, "state"},
		"phone without DDD":      {func(in *CreateInput) { in.RecipientPhone = "98765-4321" }, "recipient_phone"},
		"latitude alone":         {func(in *CreateInput) { in.Longitude = nil }, "latitude"},
		"latitude out of range":  {func(in *CreateInput) { in.Latitude = ptr(91.0) }, "latitude"},
		"longitude out of range": {func(in *CreateInput) { in.Longitude = ptr(-181.0) }, "longitude"},
		"long street":            {func(in *CreateInput) { in.Street = strings.Repeat("a", 201) }, "street"},
		"long reference":         {func(in *CreateInput) { in.AddressReference = strings.Repeat("a", 301) }, "address_reference"},
		"unknown driver":         {func(in *CreateInput) { in.DriverID = ptr(int64(99)) }, "driver_id"},
		"owner as driver":        {func(in *CreateInput) { in.DriverID = ptr(int64(1)) }, "driver_id"},
		"other carrier's driver": {func(in *CreateInput) { in.DriverID = ptr(int64(5)) }, "driver_id"},
		"long name":              {func(in *CreateInput) { in.RecipientName = strings.Repeat("a", 121) }, "recipient_name"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)
			_, err := svc.Create(context.Background(), owner, in)

			var verr *apperr.ValidationError
			require.ErrorAs(t, err, &verr)
			assert.Contains(t, verr.Fields, tc.field)
		})
	}
}

// The limits count characters, as the form's maxLength does: a name full of
// accents fits as long as a plain one.
func TestCreate_LimitsCountCharacters(t *testing.T) {
	in := validInput()
	in.RecipientName = strings.Repeat("ã", 120)
	in.Street = strings.Repeat("ç", 200)
	_, err := NewService(newFakeStore()).Create(context.Background(), owner, in)
	assert.NoError(t, err)
}

func TestCreate_WithDriver(t *testing.T) {
	svc := NewService(newFakeStore())
	in := validInput()
	in.DriverID = ptr(int64(2))

	d, err := svc.Create(context.Background(), owner, in)
	require.NoError(t, err)
	assert.Equal(t, ptr(int64(2)), d.DriverID)
}

func TestCreate_RetriesOnCodeCollision(t *testing.T) {
	fs := newFakeStore()
	fs.createErrs = []error{&pgconn.PgError{Code: "23505"}}
	svc := NewService(fs)

	_, err := svc.Create(context.Background(), owner, validInput())
	require.NoError(t, err)
	assert.Len(t, fs.deliveries, 1)
}

func TestGetAndUpdate_NotFound(t *testing.T) {
	svc := NewService(newFakeStore())

	_, err := svc.Get(context.Background(), owner, 404)
	assert.ErrorIs(t, err, apperr.ErrNotFound)

	_, err = svc.Update(context.Background(), owner, 404, UpdateInput{Number: ptr("20")})
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

// Carriers are tenants: another carrier's delivery answers like a missing one.
func TestCarrierIsolation(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)
	ctx := context.Background()
	in := validInput()
	in.DriverID = ptr(int64(2))
	d, err := svc.Create(ctx, owner, in)
	require.NoError(t, err)
	assert.Equal(t, int64(1), d.CarrierID)

	_, err = svc.Get(ctx, rival, d.ID)
	assert.ErrorIs(t, err, apperr.ErrNotFound)
	_, err = svc.Update(ctx, rival, d.ID, UpdateInput{Number: ptr("20")})
	assert.ErrorIs(t, err, apperr.ErrNotFound)
	_, err = svc.ListEvents(ctx, rival, d.ID)
	assert.ErrorIs(t, err, apperr.ErrNotFound)
	_, err = svc.AddEvent(ctx, rival, d.ID, EventInput{Status: StatusPickedUp})
	assert.ErrorIs(t, err, apperr.ErrNotFound)
	_, err = svc.AddEvent(ctx, rivalDriver, d.ID, EventInput{Status: StatusPickedUp})
	assert.ErrorIs(t, err, apperr.ErrNotFound)

	list, err := svc.List(ctx, rival.CarrierID, ListInput{})
	require.NoError(t, err)
	assert.Empty(t, list)
	_, err = svc.Update(ctx, owner, d.ID, UpdateInput{DriverID: ptr(int64(5))})
	var verr *apperr.ValidationError
	require.ErrorAs(t, err, &verr, "another carrier's driver cannot be assigned")
	assert.Contains(t, verr.Fields, "driver_id")
}

func TestSummary(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)
	ctx := context.Background()
	for range 2 {
		_, err := svc.Create(ctx, owner, validInput())
		require.NoError(t, err)
	}
	withDriver := validInput()
	withDriver.DriverID = ptr(int64(2))
	d, err := svc.Create(ctx, owner, withDriver)
	require.NoError(t, err)
	_, err = svc.AddEvent(ctx, driverA, d.ID, EventInput{Status: StatusPickedUp})
	require.NoError(t, err)
	_, err = svc.Create(ctx, rival, validInput())
	require.NoError(t, err)

	sum, err := svc.Summary(ctx, owner.CarrierID)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{
		StatusPending: 2, StatusPickedUp: 1, StatusInTransit: 0, StatusDelivered: 0, StatusFailed: 0,
	}, sum.ByStatus, "every status is present and other carriers are left out")
	assert.Equal(t, int64(2), sum.Unassigned)
}

func TestUpdate(t *testing.T) {
	svc := NewService(newFakeStore())
	created, err := svc.Create(context.Background(), owner, validInput())
	require.NoError(t, err)

	updated, err := svc.Update(context.Background(), owner, created.ID, UpdateInput{Number: ptr(" 20 "), DriverID: ptr(int64(2))})
	require.NoError(t, err)
	assert.Equal(t, "Praça da Sé, 20, Apto 2 - Sé, São Paulo - SP, 01001-000", updated.Address)
	assert.Nil(t, updated.Latitude, "a new address without coordinates drops the old map position")
	assert.Equal(t, created.RecipientName, updated.RecipientName, "fields not sent stay the same")
	assert.Equal(t, ptr(int64(2)), updated.DriverID)

	updated, err = svc.Update(context.Background(), owner, created.ID, UpdateInput{Latitude: ptr(-23.5), Longitude: ptr(-46.6)})
	require.NoError(t, err)
	assert.Equal(t, ptr(-23.5), updated.Latitude, "the pin can be moved alone")

	updated, err = svc.Update(context.Background(), owner, created.ID, UpdateInput{RecipientName: ptr("Maria Lima")})
	require.NoError(t, err)
	assert.Equal(t, ptr(-23.5), updated.Latitude, "a change outside the address keeps the map position")

	_, err = svc.Update(context.Background(), owner, created.ID, UpdateInput{RecipientEmail: ptr("nope")})
	var verr *apperr.ValidationError
	assert.ErrorAs(t, err, &verr)
}

// retentionInBetween anonymizes the delivery between the service reading it
// and writing the change, as the retention job could.
type retentionInBetween struct{ *fakeStore }

func (r retentionInBetween) UpdateDelivery(ctx context.Context, arg store.UpdateDeliveryParams) (store.Delivery, error) {
	d := r.deliveries[arg.ID]
	d.AnonymizedAt = ptr(time.Now())
	r.deliveries[arg.ID] = d
	return r.fakeStore.UpdateDelivery(ctx, arg)
}

func TestUpdate_RacingRetention(t *testing.T) {
	fs := newFakeStore()
	created, err := NewService(fs).Create(context.Background(), owner, validInput())
	require.NoError(t, err)

	_, err = NewService(retentionInBetween{fs}).Update(context.Background(), owner, created.ID, UpdateInput{Number: ptr("20")})
	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.Equal(t, "10", fs.deliveries[created.ID].Number, "nothing is written over the erased data")
}

// usersDown fails to read users, as when the database is out.
type usersDown struct{ *fakeStore }

func (usersDown) GetUserByID(context.Context, int64) (store.User, error) {
	return store.User{}, errors.New("connection refused")
}

func TestDriverCheck_DatabaseErrorIsNotInvalidInput(t *testing.T) {
	fs := newFakeStore()
	created, err := NewService(fs).Create(context.Background(), owner, validInput())
	require.NoError(t, err)
	svc := NewService(usersDown{fs})

	in := validInput()
	in.DriverID = ptr[int64](2)
	_, err = svc.Create(context.Background(), owner, in)
	var verr *apperr.ValidationError
	assert.False(t, errors.As(err, &verr), "a 500, not a 422 blaming the driver")
	assert.ErrorContains(t, err, "connection refused")

	_, err = svc.Update(context.Background(), owner, created.ID, UpdateInput{DriverID: ptr[int64](2)})
	assert.False(t, errors.As(err, &verr))
	assert.ErrorContains(t, err, "connection refused")
}

// Deliveries created before the address was split only have the one-line
// address; editing anything else must not ask for the parts.
func TestUpdate_LegacyAddress(t *testing.T) {
	fs := newFakeStore()
	fs.deliveries[7] = store.Delivery{ID: 7, CarrierID: 1, RecipientName: "Maria", RecipientEmail: "maria@example.com", Address: "Rua A, 10", Status: StatusPending}
	svc := NewService(fs)

	d, err := svc.Update(context.Background(), owner, 7, UpdateInput{RecipientName: ptr("Maria Souza")})
	require.NoError(t, err)
	assert.Equal(t, "Rua A, 10", d.Address)

	_, err = svc.Update(context.Background(), owner, 7, UpdateInput{Number: ptr("20")})
	var verr *apperr.ValidationError
	require.ErrorAs(t, err, &verr, "changing the address asks for all of it")
	assert.Contains(t, verr.Fields, "postal_code")
}

func TestList_Paging(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)

	_, err := svc.List(context.Background(), 1, ListInput{})
	require.NoError(t, err)
	assert.Equal(t, store.ListDeliveriesParams{CarrierID: 1, Limit: 20, Offset: 0}, fs.lastList)

	_, err = svc.List(context.Background(), 1, ListInput{Page: 3, Size: 10, Status: ptr(StatusDelivered)})
	require.NoError(t, err)
	assert.Equal(t, store.ListDeliveriesParams{CarrierID: 1, Status: ptr(StatusDelivered), Limit: 10, Offset: 20}, fs.lastList)

	_, err = svc.List(context.Background(), 1, ListInput{Size: 1000})
	require.NoError(t, err)
	assert.Equal(t, int32(20), fs.lastList.Limit, "oversized pages fall back to the default")

	_, err = svc.List(context.Background(), 1, ListInput{Status: ptr("lost")})
	var verr *apperr.ValidationError
	assert.ErrorAs(t, err, &verr)

	_, err = svc.List(context.Background(), 1, ListInput{Page: 999_999_999})
	require.ErrorAs(t, err, &verr, "a huge page would overflow the offset")
	assert.Contains(t, verr.Fields, "page")
}

func TestList_DriverFilter(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)

	_, err := svc.List(context.Background(), 1, ListInput{Driver: NoDriver})
	require.NoError(t, err)
	assert.True(t, fs.lastList.Unassigned)
	assert.Nil(t, fs.lastList.DriverID)

	_, err = svc.List(context.Background(), 1, ListInput{Driver: "7"})
	require.NoError(t, err)
	assert.False(t, fs.lastList.Unassigned)
	assert.Equal(t, ptr(int64(7)), fs.lastList.DriverID)

	for _, bad := range []string{"nobody", "0", "-3", "1.5", "99999999999999999999"} {
		_, err = svc.List(context.Background(), 1, ListInput{Driver: bad})
		var verr *apperr.ValidationError
		require.ErrorAs(t, err, &verr, bad)
		assert.Contains(t, verr.Fields, "driver", bad)
	}
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
