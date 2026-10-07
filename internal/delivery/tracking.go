package delivery

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// trackingTTL is how long the public link keeps working after the delivery
// is delivered or last failed.
const trackingTTL = 30 * 24 * time.Hour

var trackingCodeRe = regexp.MustCompile(`^RS[` + codeAlphabet + `]{10}$`)

// Tracking is what anyone holding the tracking code may see. It leaves out
// the recipient's e-mail, address and last name, the driver and event notes.
type Tracking struct {
	TrackingCode string `json:"tracking_code"`
	// CarrierName says who is delivering, as on a shipping label.
	CarrierName        string          `json:"carrier_name"`
	Status             string          `json:"status"`
	RecipientFirstName string          `json:"recipient_first_name"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Events             []TrackingEvent `json:"events"`
}

type TrackingEvent struct {
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Track returns the public view of a delivery. Malformed, unknown and expired
// codes all answer not found.
func (s *Service) Track(ctx context.Context, code string) (Tracking, error) {
	d, err := s.PublicDelivery(ctx, code)
	if err != nil {
		return Tracking{}, err
	}
	events, err := s.store.ListDeliveryEvents(ctx, d.ID)
	if err != nil {
		return Tracking{}, err
	}
	carrier, err := s.store.GetCarrier(ctx, d.CarrierID)
	if err != nil {
		return Tracking{}, err
	}
	t := Tracking{
		TrackingCode:       d.TrackingCode,
		CarrierName:        carrier.Name,
		Status:             d.Status,
		RecipientFirstName: firstName(d.RecipientName),
		UpdatedAt:          d.UpdatedAt,
		Events:             make([]TrackingEvent, len(events)),
	}
	for i, e := range events {
		t.Events[i] = TrackingEvent{Status: e.Status, CreatedAt: e.CreatedAt}
	}
	return t, nil
}

// PublicDelivery finds the delivery behind a public tracking code, with the
// same not found as Track for malformed, unknown, expired and anonymized
// codes.
func (s *Service) PublicDelivery(ctx context.Context, code string) (store.Delivery, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !IsTrackingCode(code) {
		return store.Delivery{}, apperr.ErrNotFound
	}
	d, err := s.store.GetDeliveryByTrackingCode(ctx, code)
	if err != nil {
		return store.Delivery{}, notFound(err)
	}
	if d.AnonymizedAt != nil || (d.CompletedAt != nil && s.now().Sub(*d.CompletedAt) > trackingTTL) {
		return store.Delivery{}, apperr.ErrNotFound
	}
	return d, nil
}

// IsTrackingCode reports whether code has the shape of a tracking code, so
// what is not one is refused without a query.
func IsTrackingCode(code string) bool {
	return trackingCodeRe.MatchString(code)
}

func firstName(name string) string {
	if fields := strings.Fields(name); len(fields) > 0 {
		return fields[0]
	}
	return ""
}
