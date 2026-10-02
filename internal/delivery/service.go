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
}

func fromStore(d store.Delivery) Delivery {
	return Delivery(d)
}

type CreateInput struct {
	RecipientName  string `json:"recipient_name"`
	RecipientEmail string `json:"recipient_email"`
	Address        string `json:"address"`
	DriverID       *int64 `json:"driver_id"`
}

// UpdateInput only changes the fields that are sent. The status is not
// editable here: it changes through delivery events.
type UpdateInput struct {
	RecipientName  *string `json:"recipient_name"`
	RecipientEmail *string `json:"recipient_email"`
	Address        *string `json:"address"`
	DriverID       *int64  `json:"driver_id"`
}

type ListInput struct {
	Status *string
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
	if err := s.validateCreate(ctx, actor.CarrierID, &in); err != nil {
		return Delivery{}, err
	}
	var out Delivery
	err := s.retryOnCodeCollision(ctx, func(q Store) (err error) {
		out, err = s.insert(ctx, q, actor, in)
		return err
	})
	if err == nil {
		s.announce(ctx, out.CarrierID, out.ID, out.TrackingCode, out.Status, false)
	}
	return out, err
}

func (s *Service) validateCreate(ctx context.Context, carrierID int64, in *CreateInput) error {
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.RecipientEmail = strings.ToLower(strings.TrimSpace(in.RecipientEmail))
	in.Address = strings.TrimSpace(in.Address)

	v := apperr.Validator{}
	v.Check(in.RecipientName != "", "recipient_name", "is required")
	v.Check(validEmail(in.RecipientEmail), "recipient_email", "must be a valid e-mail")
	v.Check(in.Address != "", "address", "is required")
	checkLengths(v, &in.RecipientName, &in.RecipientEmail, &in.Address)
	if in.DriverID != nil {
		v.Check(s.isDriver(ctx, carrierID, *in.DriverID), "driver_id", "must be an existing driver")
	}
	return v.Err()
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

func (s *Service) insert(ctx context.Context, q Store, actor auth.Claims, in CreateInput) (Delivery, error) {
	code, err := s.newCode()
	if err != nil {
		return Delivery{}, err
	}
	d, err := q.CreateDelivery(ctx, store.CreateDeliveryParams{
		CarrierID:      actor.CarrierID,
		TrackingCode:   code,
		RecipientName:  in.RecipientName,
		RecipientEmail: in.RecipientEmail,
		Address:        in.Address,
		DriverID:       in.DriverID,
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

// List lists the carrier's deliveries; the filter is part of the query.
func (s *Service) List(ctx context.Context, carrierID int64, in ListInput) ([]Delivery, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	limit, offset := in.page()
	rows, err := s.store.ListDeliveries(ctx, store.ListDeliveriesParams{
		CarrierID: carrierID, Status: in.Status, Limit: limit, Offset: offset,
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
	v := apperr.Validator{}
	if in.RecipientName != nil {
		*in.RecipientName = strings.TrimSpace(*in.RecipientName)
		v.Check(*in.RecipientName != "", "recipient_name", "cannot be empty")
	}
	if in.RecipientEmail != nil {
		*in.RecipientEmail = strings.ToLower(strings.TrimSpace(*in.RecipientEmail))
		v.Check(validEmail(*in.RecipientEmail), "recipient_email", "must be a valid e-mail")
	}
	if in.Address != nil {
		*in.Address = strings.TrimSpace(*in.Address)
		v.Check(*in.Address != "", "address", "cannot be empty")
	}
	checkLengths(v, in.RecipientName, in.RecipientEmail, in.Address)
	if in.DriverID != nil {
		v.Check(s.isDriver(ctx, actor.CarrierID, *in.DriverID), "driver_id", "must be an existing driver")
	}
	if err := v.Err(); err != nil {
		return Delivery{}, err
	}
	cur, err := s.visible(ctx, actor, id)
	if err != nil {
		return Delivery{}, err
	}
	if cur.AnonymizedAt != nil {
		return Delivery{}, errAnonymized
	}

	d, err := s.store.UpdateDelivery(ctx, store.UpdateDeliveryParams{
		ID:             id,
		RecipientName:  in.RecipientName,
		RecipientEmail: in.RecipientEmail,
		Address:        in.Address,
		DriverID:       in.DriverID,
	})
	if err != nil {
		return Delivery{}, notFound(err)
	}
	// The public page shows the recipient's first name, so it reloads too.
	s.announce(ctx, d.CarrierID, d.ID, d.TrackingCode, d.Status, in.RecipientName != nil)
	return fromStore(d), nil
}

const (
	maxName    = 120
	maxEmail   = 254
	maxAddress = 300
)

// checkLengths caps free-text fields; nil means the field was not sent.
func checkLengths(v apperr.Validator, name, email, address *string) {
	if name != nil {
		v.Check(len(*name) <= maxName, "recipient_name", "must have at most 120 characters")
	}
	if email != nil {
		v.Check(len(*email) <= maxEmail, "recipient_email", "must be a valid e-mail")
	}
	if address != nil {
		v.Check(len(*address) <= maxAddress, "address", "must have at most 300 characters")
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
