// Package delivery handles creating, listing and editing deliveries.
package delivery

import (
	"context"
	"crypto/rand"
	"errors"
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
	ListDeliveries(ctx context.Context, arg store.ListDeliveriesParams) ([]store.Delivery, error)
	UpdateDelivery(ctx context.Context, arg store.UpdateDeliveryParams) (store.Delivery, error)
	GetUserByID(ctx context.Context, id int64) (store.User, error)
}

type Delivery struct {
	ID             int64     `json:"id"`
	TrackingCode   string    `json:"tracking_code"`
	RecipientName  string    `json:"recipient_name"`
	RecipientEmail string    `json:"recipient_email"`
	Address        string    `json:"address"`
	Status         string    `json:"status"`
	DriverID       *int64    `json:"driver_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
}

func NewService(s Store) *Service {
	return &Service{store: s, newCode: NewTrackingCode}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Delivery, error) {
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.RecipientEmail = strings.ToLower(strings.TrimSpace(in.RecipientEmail))
	in.Address = strings.TrimSpace(in.Address)

	v := apperr.Validator{}
	v.Check(in.RecipientName != "", "recipient_name", "is required")
	v.Check(validEmail(in.RecipientEmail), "recipient_email", "must be a valid e-mail")
	v.Check(in.Address != "", "address", "is required")
	if in.DriverID != nil {
		v.Check(s.isDriver(ctx, *in.DriverID), "driver_id", "must be an existing driver")
	}
	if err := v.Err(); err != nil {
		return Delivery{}, err
	}

	// A collision between random codes is very unlikely, but retrying is cheap.
	for range 3 {
		code, err := s.newCode()
		if err != nil {
			return Delivery{}, err
		}
		d, err := s.store.CreateDelivery(ctx, store.CreateDeliveryParams{
			TrackingCode:   code,
			RecipientName:  in.RecipientName,
			RecipientEmail: in.RecipientEmail,
			Address:        in.Address,
			DriverID:       in.DriverID,
		})
		if isUniqueViolation(err) {
			continue
		}
		if err != nil {
			return Delivery{}, err
		}
		return fromStore(d), nil
	}
	return Delivery{}, errors.New("could not generate a unique tracking code")
}

func (s *Service) Get(ctx context.Context, id int64) (Delivery, error) {
	d, err := s.store.GetDelivery(ctx, id)
	if err != nil {
		return Delivery{}, notFound(err)
	}
	return fromStore(d), nil
}

func (s *Service) List(ctx context.Context, in ListInput) ([]Delivery, error) {
	v := apperr.Validator{}
	if in.Status != nil {
		v.Check(validStatus(*in.Status), "status", "must be one of "+strings.Join(statuses, ", "))
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	page, size := max(in.Page, 1), in.Size
	if size <= 0 || size > 100 {
		size = 20
	}
	rows, err := s.store.ListDeliveries(ctx, store.ListDeliveriesParams{
		Status: in.Status,
		Limit:  int32(size),
		Offset: int32((page - 1) * size),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Delivery, len(rows))
	for i, d := range rows {
		out[i] = fromStore(d)
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (Delivery, error) {
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
	if in.DriverID != nil {
		v.Check(s.isDriver(ctx, *in.DriverID), "driver_id", "must be an existing driver")
	}
	if err := v.Err(); err != nil {
		return Delivery{}, err
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
	return fromStore(d), nil
}

func (s *Service) isDriver(ctx context.Context, id int64) bool {
	u, err := s.store.GetUserByID(ctx, id)
	return err == nil && u.Role == auth.RoleDriver
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
