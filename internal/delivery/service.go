// Package delivery handles deliveries: creating and editing them, their
// status events and the public tracking page.
package delivery

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

const (
	StatusPending   = "pending"
	StatusPickedUp  = "picked_up"
	StatusInTransit = "in_transit"
	StatusDelivered = "delivered"
	StatusFailed    = "failed"
)

var statuses = []string{StatusPending, StatusPickedUp, StatusInTransit, StatusDelivered, StatusFailed}

type Store interface {
	CreateDelivery(ctx context.Context, arg store.CreateDeliveryParams) (store.Delivery, error)
	GetDelivery(ctx context.Context, id int64) (store.Delivery, error)
	GetDeliveryByTrackingCode(ctx context.Context, trackingCode string) (store.Delivery, error)
	ListDeliveries(ctx context.Context, arg store.ListDeliveriesParams) ([]store.Delivery, error)
	CountDeliveriesByStatus(ctx context.Context, arg store.CountDeliveriesByStatusParams) ([]store.CountDeliveriesByStatusRow, error)
	CountUnassignedDeliveries(ctx context.Context, carrierID int64) (int64, error)
	ListDriverDeliveries(ctx context.Context, arg store.ListDriverDeliveriesParams) ([]store.Delivery, error)
	UpdateDelivery(ctx context.Context, arg store.UpdateDeliveryParams) (store.Delivery, error)
	SetDeliveryStatus(ctx context.Context, arg store.SetDeliveryStatusParams) (store.Delivery, error)
	CreateDeliveryEvent(ctx context.Context, arg store.CreateDeliveryEventParams) (store.DeliveryEvent, error)
	ListDeliveryEvents(ctx context.Context, deliveryID int64) ([]store.DeliveryEvent, error)
	DeleteExpiredIdempotencyKey(ctx context.Context, arg store.DeleteExpiredIdempotencyKeyParams) error
	ReserveIdempotencyKey(ctx context.Context, arg store.ReserveIdempotencyKeyParams) (store.IdempotencyKey, error)
	GetIdempotencyKey(ctx context.Context, arg store.GetIdempotencyKeyParams) (store.IdempotencyKey, error)
	SetIdempotencyKeyDelivery(ctx context.Context, arg store.SetIdempotencyKeyDeliveryParams) error
	GetUserByID(ctx context.Context, id int64) (store.User, error)
	GetCarrier(ctx context.Context, id int64) (store.Carrier, error)
	// InTx runs fn in a database transaction, passing a Store bound to it.
	InTx(ctx context.Context, fn func(Store) error) error
}

type Delivery struct {
	ID             int64      `json:"id"`
	TrackingCode   string     `json:"tracking_code"`
	RecipientName  string     `json:"recipient_name"`
	RecipientEmail string     `json:"recipient_email"`
	Address        string     `json:"address"`
	Status         string     `json:"status"`
	DriverID       *int64     `json:"driver_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	// AnonymizedAt is set when the recipient's data was erased (see
	// internal/retention); such a delivery can no longer change.
	AnonymizedAt *time.Time `json:"anonymized_at"`
	// CarrierID is the carrier that owns the delivery; callers only ever
	// see their own carrier's deliveries, so it is not sent.
	CarrierID int64 `json:"-"`
	// RecipientPhone and the address parts are empty for deliveries
	// created before the address was split; those only have Address.
	RecipientPhone   string   `json:"recipient_phone"`
	PostalCode       string   `json:"postal_code"`
	Street           string   `json:"street"`
	Number           string   `json:"number"`
	Complement       string   `json:"complement"`
	District         string   `json:"district"`
	City             string   `json:"city"`
	State            string   `json:"state"`
	AddressReference string   `json:"address_reference"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
}

func fromStore(d store.Delivery) Delivery {
	return Delivery(d)
}

// CreateInput is the recipient and the address in parts; the one-line
// address is built from them. Latitude and longitude are optional (the
// carrier's form places the address on a map) and go together.
type CreateInput struct {
	RecipientName    string   `json:"recipient_name"`
	RecipientEmail   string   `json:"recipient_email"`
	RecipientPhone   string   `json:"recipient_phone"`
	PostalCode       string   `json:"postal_code"`
	Street           string   `json:"street"`
	Number           string   `json:"number"`
	Complement       string   `json:"complement"`
	District         string   `json:"district"`
	City             string   `json:"city"`
	State            string   `json:"state"`
	AddressReference string   `json:"address_reference"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	DriverID         *int64   `json:"driver_id"`
}

func (in CreateInput) recipient() recipient {
	return recipient{
		Name: in.RecipientName, Email: in.RecipientEmail, Phone: in.RecipientPhone,
		PostalCode: in.PostalCode, Street: in.Street, Number: in.Number, Complement: in.Complement,
		District: in.District, City: in.City, State: in.State, Reference: in.AddressReference,
		Latitude: in.Latitude, Longitude: in.Longitude,
	}
}

// UpdateInput only changes the fields that are sent. Changing any part of
// the address checks the whole address again, and the map position is
// dropped unless new coordinates come with it. The status is not editable
// here: it changes through delivery events.
type UpdateInput struct {
	RecipientName    *string  `json:"recipient_name"`
	RecipientEmail   *string  `json:"recipient_email"`
	RecipientPhone   *string  `json:"recipient_phone"`
	PostalCode       *string  `json:"postal_code"`
	Street           *string  `json:"street"`
	Number           *string  `json:"number"`
	Complement       *string  `json:"complement"`
	District         *string  `json:"district"`
	City             *string  `json:"city"`
	State            *string  `json:"state"`
	AddressReference *string  `json:"address_reference"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	DriverID         *int64   `json:"driver_id"`
}

// apply merges the sent fields into r and tells whether the address changed.
func (in UpdateInput) apply(r *recipient) (addressChanged bool) {
	set := func(dst *string, src *string) bool {
		if src != nil {
			*dst = *src
		}
		return src != nil
	}
	set(&r.Name, in.RecipientName)
	set(&r.Email, in.RecipientEmail)
	set(&r.Phone, in.RecipientPhone)
	for _, f := range []struct {
		dst *string
		src *string
	}{
		{&r.PostalCode, in.PostalCode}, {&r.Street, in.Street}, {&r.Number, in.Number},
		{&r.Complement, in.Complement}, {&r.District, in.District}, {&r.City, in.City},
		{&r.State, in.State}, {&r.Reference, in.AddressReference},
	} {
		addressChanged = set(f.dst, f.src) || addressChanged
	}
	if in.Latitude != nil || in.Longitude != nil {
		r.Latitude, r.Longitude = in.Latitude, in.Longitude
		addressChanged = true
	} else if addressChanged {
		r.Latitude, r.Longitude = nil, nil
	}
	return addressChanged
}

type ListInput struct {
	Status *string
	// Search looks for part of the tracking code or of the recipient's
	// name or e-mail, ignoring case and accents. Only the carrier's list
	// uses it.
	Search string
	Page   int
	Size   int
}

type Service struct {
	store   Store
	newCode func() (string, error)
	now     func() time.Time
	pub     Publisher
}

func NewService(s Store) *Service {
	return &Service{store: s, newCode: NewTrackingCode, now: time.Now}
}

// Create adds a delivery to the actor's carrier, with its first "pending"
// event recorded as made by the actor.
func (s *Service) Create(ctx context.Context, actor auth.Claims, in CreateInput) (Delivery, error) {
	rec, err := s.validateCreate(ctx, actor.CarrierID, in)
	if err != nil {
		return Delivery{}, err
	}
	var out Delivery
	err = s.retryOnCodeCollision(ctx, func(q Store) (err error) {
		out, err = s.insert(ctx, q, actor, rec, in.DriverID)
		return err
	})
	if err == nil {
		s.announce(ctx, out.CarrierID, out.ID, out.TrackingCode, out.Status, false)
	}
	return out, err
}

func (s *Service) validateCreate(ctx context.Context, carrierID int64, in CreateInput) (recipient, error) {
	rec := in.recipient()
	rec.normalize()
	v := apperr.Validator{}
	rec.validateContact(v)
	rec.validatePhone(v)
	rec.validateAddress(v)
	if in.DriverID != nil {
		v.Check(s.isDriver(ctx, carrierID, *in.DriverID), "driver_id", "must be an existing driver")
	}
	return rec, v.Err()
}

// retryOnCodeCollision runs fn in a transaction, starting over with a new
// one when a tracking code is already taken. A collision between random
// codes is very unlikely, but retrying is cheap; it cannot happen inside the
// same transaction because Postgres aborts it on the error.
func (s *Service) retryOnCodeCollision(ctx context.Context, fn func(q Store) error) error {
	for range 3 {
		err := s.store.InTx(ctx, fn)
		if isUniqueViolation(err) {
			continue
		}
		return err
	}
	return errors.New("could not generate a unique tracking code")
}

func (s *Service) insert(ctx context.Context, q Store, actor auth.Claims, r recipient, driverID *int64) (Delivery, error) {
	code, err := s.newCode()
	if err != nil {
		return Delivery{}, err
	}
	d, err := q.CreateDelivery(ctx, store.CreateDeliveryParams{
		CarrierID:        actor.CarrierID,
		TrackingCode:     code,
		RecipientName:    r.Name,
		RecipientEmail:   r.Email,
		RecipientPhone:   r.Phone,
		Address:          r.fullAddress(),
		PostalCode:       r.PostalCode,
		Street:           r.Street,
		Number:           r.Number,
		Complement:       r.Complement,
		District:         r.District,
		City:             r.City,
		State:            r.State,
		AddressReference: r.Reference,
		Latitude:         r.Latitude,
		Longitude:        r.Longitude,
		DriverID:         driverID,
	})
	if err != nil {
		return Delivery{}, err
	}
	_, err = q.CreateDeliveryEvent(ctx, store.CreateDeliveryEventParams{
		DeliveryID: d.ID, Status: d.Status, CreatedBy: &actor.UserID,
	})
	if err != nil {
		return Delivery{}, err
	}
	return fromStore(d), nil
}

// Get returns a delivery of the actor's carrier; any other is not found.
func (s *Service) Get(ctx context.Context, actor auth.Claims, id int64) (Delivery, error) {
	d, err := s.visible(ctx, actor, id)
	if err != nil {
		return Delivery{}, err
	}
	return fromStore(d), nil
}

// List lists the carrier's deliveries; the filters are part of the query.
func (s *Service) List(ctx context.Context, carrierID int64, in ListInput) ([]Delivery, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	limit, offset := in.page()
	rows, err := s.store.ListDeliveries(ctx, store.ListDeliveriesParams{
		CarrierID: carrierID, Status: in.Status, Search: searchPattern(in.Search), Limit: limit, Offset: offset,
	})
	return fromStoreList(rows), err
}

// ListForDriver lists only the deliveries assigned to driverID; the filter
// is part of the query, so other drivers' deliveries never leave the database.
func (s *Service) ListForDriver(ctx context.Context, driverID int64, in ListInput) ([]Delivery, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	limit, offset := in.page()
	rows, err := s.store.ListDriverDeliveries(ctx, store.ListDriverDeliveriesParams{
		DriverID: driverID, Status: in.Status, Limit: limit, Offset: offset,
	})
	return fromStoreList(rows), err
}

func (in ListInput) validate() error {
	v := apperr.Validator{}
	if in.Status != nil {
		v.Check(validStatus(*in.Status), "status", "must be one of "+strings.Join(statuses, ", "))
	}
	v.Check(in.Page <= maxPage, "page", "must be at most 10000")
	v.Check(utf8.RuneCountInString(in.Search) <= maxSearch, "q", "must have at most 100 characters")
	return v.Err()
}

// maxPage keeps the offset (page * size) far from int32 overflow.
const maxPage = 10_000

// page turns page/size into limit/offset, falling back to 20 items per page.
func (in ListInput) page() (limit, offset int32) {
	page, size := min(max(in.Page, 1), maxPage), in.Size
	if size <= 0 || size > 100 {
		size = 20
	}
	// Both are bounded above (size <= 100, page <= maxPage), so they fit in int32.
	return int32(size), int32((page - 1) * size) //nolint:gosec // bounded above

}

var errAnonymized = fmt.Errorf("%w: the recipient's data was erased, the delivery can no longer change", apperr.ErrConflict)

func fromStoreList(rows []store.Delivery) []Delivery {
	out := make([]Delivery, len(rows))
	for i, d := range rows {
		out[i] = fromStore(d)
	}
	return out
}

func (s *Service) Update(ctx context.Context, actor auth.Claims, id int64, in UpdateInput) (Delivery, error) {
	cur, err := s.visible(ctx, actor, id)
	if err != nil {
		return Delivery{}, err
	}
	if cur.AnonymizedAt != nil {
		return Delivery{}, errAnonymized
	}
	rec := recipientOf(cur)
	addressChanged := in.apply(&rec)
	rec.normalize()

	v := apperr.Validator{}
	rec.validateContact(v)
	if in.RecipientPhone != nil {
		rec.validatePhone(v)
	}
	if addressChanged {
		rec.validateAddress(v)
	}
	if in.DriverID != nil {
		v.Check(s.isDriver(ctx, actor.CarrierID, *in.DriverID), "driver_id", "must be an existing driver")
	}
	if err := v.Err(); err != nil {
		return Delivery{}, err
	}
	address := cur.Address
	if addressChanged {
		address = rec.fullAddress()
	}

	d, err := s.store.UpdateDelivery(ctx, store.UpdateDeliveryParams{
		ID:               id,
		RecipientName:    rec.Name,
		RecipientEmail:   rec.Email,
		RecipientPhone:   rec.Phone,
		Address:          address,
		PostalCode:       rec.PostalCode,
		Street:           rec.Street,
		Number:           rec.Number,
		Complement:       rec.Complement,
		District:         rec.District,
		City:             rec.City,
		State:            rec.State,
		AddressReference: rec.Reference,
		Latitude:         rec.Latitude,
		Longitude:        rec.Longitude,
		DriverID:         in.DriverID,
	})
	if err != nil {
		return Delivery{}, notFound(err)
	}
	// The public page shows the recipient's first name, so it reloads too.
	s.announce(ctx, d.CarrierID, d.ID, d.TrackingCode, d.Status, in.RecipientName != nil)
	return fromStore(d), nil
}

func recipientOf(d store.Delivery) recipient {
	return recipient{
		Name: d.RecipientName, Email: d.RecipientEmail, Phone: d.RecipientPhone,
		PostalCode: d.PostalCode, Street: d.Street, Number: d.Number, Complement: d.Complement,
		District: d.District, City: d.City, State: d.State, Reference: d.AddressReference,
		Latitude: d.Latitude, Longitude: d.Longitude,
	}
}

// isDriver tells whether id is a driver of the carrier; another carrier's
// driver gets the same answer as a missing one.
func (s *Service) isDriver(ctx context.Context, carrierID, id int64) bool {
	u, err := s.store.GetUserByID(ctx, id)
	return err == nil && u.Role == auth.RoleDriver && u.CarrierID == carrierID
}

// Tracking codes skip 0/O and 1/I so they are easy to read over the phone.
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// NewTrackingCode returns a code like "RS7K2M9QXA4P".
func NewTrackingCode() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = codeAlphabet[int(b)%len(codeAlphabet)]
	}
	return "RS" + string(buf), nil
}

func validStatus(s string) bool {
	return slices.Contains(statuses, s)
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
