package delivery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// transitions lists where each status can go. delivered is final; failed can
// go back in transit for a new attempt.
var transitions = map[string][]string{
	StatusPending:   {StatusPickedUp, StatusFailed},
	StatusPickedUp:  {StatusInTransit, StatusFailed},
	StatusInTransit: {StatusDelivered, StatusFailed},
	StatusFailed:    {StatusInTransit},
}

const maxNote = 500

type Event struct {
	ID         int64     `json:"id"`
	DeliveryID int64     `json:"delivery_id"`
	Status     string    `json:"status"`
	Note       *string   `json:"note"`
	CreatedBy  *int64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

func eventFromStore(e store.DeliveryEvent) Event {
	return Event{
		ID: e.ID, DeliveryID: e.DeliveryID, Status: e.Status,
		Note: e.Note, CreatedBy: e.CreatedBy, CreatedAt: e.CreatedAt,
	}
}

type EventInput struct {
	Status string  `json:"status"`
	Note   *string `json:"note"`
}

// AddEvent moves a delivery to a new status and records who did it. Drivers
// may only touch deliveries assigned to them; for any other they get the
// same not found as for a missing delivery.
func (s *Service) AddEvent(ctx context.Context, actor auth.Claims, deliveryID int64, in EventInput) (Event, error) {
	if in.Note != nil {
		if *in.Note = strings.TrimSpace(*in.Note); *in.Note == "" {
			in.Note = nil
		}
	}
	v := apperr.Validator{}
	v.Check(validStatus(in.Status), "status", "must be one of "+strings.Join(statuses, ", "))
	v.Check(in.Status != StatusFailed || in.Note != nil, "note", "is required when the delivery fails")
	v.Check(in.Note == nil || utf8.RuneCountInString(*in.Note) <= maxNote, "note", "must have at most 500 characters")
	if err := v.Err(); err != nil {
		return Event{}, err
	}

	d, err := s.visible(ctx, actor, deliveryID)
	if err != nil {
		return Event{}, err
	}
	if d.AnonymizedAt != nil {
		return Event{}, errAnonymized
	}
	if !slices.Contains(transitions[d.Status], in.Status) {
		return Event{}, fmt.Errorf("%w: cannot go from %s to %s", apperr.ErrConflict, d.Status, in.Status)
	}
	var completedAt *time.Time
	if in.Status == StatusDelivered || in.Status == StatusFailed {
		now := s.now()
		completedAt = &now
	}

	var ev store.DeliveryEvent
	err = s.store.InTx(ctx, func(q Store) error {
		_, err := q.SetDeliveryStatus(ctx, store.SetDeliveryStatusParams{
			ID: deliveryID, FromStatus: d.Status, Status: in.Status, CompletedAt: completedAt,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: the delivery status changed, reload and try again", apperr.ErrConflict)
		}
		if err != nil {
			return err
		}
		ev, err = q.CreateDeliveryEvent(ctx, store.CreateDeliveryEventParams{
			DeliveryID: deliveryID, Status: in.Status, Note: in.Note, CreatedBy: &actor.UserID,
		})
		return err
	})
	if err != nil {
		return Event{}, err
	}
	s.announce(ctx, d.CarrierID, deliveryID, d.TrackingCode, in.Status, true)
	return eventFromStore(ev), nil
}

// ListEvents returns the status history of a delivery, oldest first.
func (s *Service) ListEvents(ctx context.Context, actor auth.Claims, deliveryID int64) ([]Event, error) {
	if _, err := s.visible(ctx, actor, deliveryID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListDeliveryEvents(ctx, deliveryID)
	if err != nil {
		return nil, err
	}
	out := make([]Event, len(rows))
	for i, e := range rows {
		out[i] = eventFromStore(e)
	}
	return out, nil
}

// visible loads a delivery the actor is allowed to see: a carrier sees its
// own deliveries, a driver only the ones assigned to them. Anything else
// gets the same not found as a missing delivery.
func (s *Service) visible(ctx context.Context, actor auth.Claims, id int64) (store.Delivery, error) {
	d, err := s.store.GetDelivery(ctx, id)
	if err != nil {
		return store.Delivery{}, notFound(err)
	}
	if d.CarrierID != actor.CarrierID {
		return store.Delivery{}, apperr.ErrNotFound
	}
	if actor.Role != auth.RoleCarrier && (d.DriverID == nil || *d.DriverID != actor.UserID) {
		return store.Delivery{}, apperr.ErrNotFound
	}
	return d, nil
}
