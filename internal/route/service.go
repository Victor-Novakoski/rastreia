// Package route handles the driver's route of the day: the packages loaded
// by scanning their codes, grouped into stops, in the order of delivery.
package route

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	_ "time/tzdata" // the route's day is Brazil's, whatever the server's zone

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type Store interface {
	EnsureRoute(ctx context.Context, arg store.EnsureRouteParams) (store.Route, error)
	ListRouteItems(ctx context.Context, routeID int64) ([]store.ListRouteItemsRow, error)
	AddRouteItem(ctx context.Context, arg store.AddRouteItemParams) (int64, error)
	RemoveRouteItem(ctx context.Context, arg store.RemoveRouteItemParams) (int64, error)
	SetRoutePositions(ctx context.Context, arg store.SetRoutePositionsParams) error
	ClaimDelivery(ctx context.Context, arg store.ClaimDeliveryParams) (store.Delivery, error)
	GetDeliveryByTrackingCode(ctx context.Context, trackingCode string) (store.Delivery, error)
	// InTx runs fn in a database transaction, passing a Store bound to it.
	InTx(ctx context.Context, fn func(Store) error) error
}

// Announcer tells the carrier's panel that a delivery changed.
type Announcer interface {
	Announce(ctx context.Context, d delivery.Delivery)
}

// Route is a driver's route for one day.
type Route struct {
	Date          string `json:"date"`
	TotalPackages int    `json:"total_packages"`
	Stops         []Stop `json:"stops"`
}

// Stop is one address of the route; packages to the same address are
// delivered together.
type Stop struct {
	Number    int       `json:"number"`
	Address   string    `json:"address"`
	Latitude  *float64  `json:"latitude"`
	Longitude *float64  `json:"longitude"`
	Packages  []Package `json:"packages"`
	location  *Point    // set when the stop is on the map
	items     []int64   // delivery ids, in order
}

// Package is a delivery in the route. Position numbers the packages 1..N in
// delivery order; it is not called number, which is the street number.
type Package struct {
	Position int `json:"position"`
	delivery.Delivery
}

// maxPackages caps a route; a day has far fewer.
const maxPackages = 300

var brazil = mustLoadLocation("America/Sao_Paulo")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type Service struct {
	store     Store
	announcer Announcer
	now       func() time.Time
}

func NewService(s Store, a Announcer) *Service {
	return &Service{store: s, announcer: a, now: time.Now}
}

// Today returns the driver's route for today, empty if nothing was scanned.
func (s *Service) Today(ctx context.Context, driver auth.Claims) (Route, error) {
	r, err := s.ensure(ctx, s.store, driver)
	if err != nil {
		return Route{}, err
	}
	return s.load(ctx, s.store, r)
}

var (
	errDelivered     = fmt.Errorf("%w: the delivery is already finished", apperr.ErrConflict)
	errOtherDriver   = fmt.Errorf("%w: the delivery is assigned to another driver", apperr.ErrConflict)
	errRouteTooLarge = fmt.Errorf("%w: the route already has %d packages", apperr.ErrConflict, maxPackages)
)

// Add loads a package into today's route by its tracking code, as read
// from the QR code on the label (the code or the tracking link) or typed.
// A delivery of the carrier without a driver becomes the driver's; one of
// another driver, another carrier or already delivered is refused.
func (s *Service) Add(ctx context.Context, driver auth.Claims, code string) (Route, error) {
	code = parseCode(code)
	var (
		out     Route
		claimed *store.Delivery
	)
	err := s.store.InTx(ctx, func(q Store) error {
		d, err := q.GetDeliveryByTrackingCode(ctx, code)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && d.CarrierID != driver.CarrierID) {
			return apperr.ErrNotFound
		}
		if err != nil {
			return err
		}
		if d.AnonymizedAt != nil || d.Status == delivery.StatusDelivered {
			return errDelivered
		}
		switch {
		case d.DriverID == nil:
			d, err = q.ClaimDelivery(ctx, store.ClaimDeliveryParams{ID: d.ID, DriverID: &driver.UserID})
			if errors.Is(err, pgx.ErrNoRows) {
				return errOtherDriver // another driver took it first
			}
			if err != nil {
				return err
			}
			claimed = &d
		case *d.DriverID != driver.UserID:
			return errOtherDriver
		}

		r, err := s.ensure(ctx, q, driver)
		if err != nil {
			return err
		}
		items, err := q.ListRouteItems(ctx, r.ID)
		if err != nil {
			return err
		}
		if len(items) >= maxPackages {
			return errRouteTooLarge
		}
		if _, err := q.AddRouteItem(ctx, store.AddRouteItemParams{RouteID: r.ID, DeliveryID: d.ID}); err != nil {
			return err
		}
		out, err = s.load(ctx, q, r)
		return err
	})
	if err == nil && claimed != nil && s.announcer != nil {
		s.announcer.Announce(ctx, delivery.Delivery(*claimed))
	}
	return out, err
}

// parseCode takes the tracking code out of what the scanner read: the label's
// QR code holds the public tracking link, ending in the code.
func parseCode(s string) string {
	s = strings.TrimSpace(s)
	if u, err := url.Parse(s); err == nil && u.Scheme != "" {
		s = u.Path[strings.LastIndex(u.Path, "/")+1:]
	}
	return strings.ToUpper(s)
}

// Remove takes a package out of today's route. It stays assigned to the driver.
func (s *Service) Remove(ctx context.Context, driver auth.Claims, deliveryID int64) (Route, error) {
	var out Route
	err := s.store.InTx(ctx, func(q Store) error {
		r, err := s.ensure(ctx, q, driver)
		if err != nil {
			return err
		}
		n, err := q.RemoveRouteItem(ctx, store.RemoveRouteItemParams{RouteID: r.ID, DeliveryID: deliveryID})
		if err != nil {
			return err
		}
		if n == 0 {
			return apperr.ErrNotFound
		}
		out, err = s.load(ctx, q, r)
		return err
	})
	return out, err
}

// Reorder saves the order the driver chose. ids lists every package of the
// route once; packages of the same stop stay together, in the place of the
// first of them.
func (s *Service) Reorder(ctx context.Context, driver auth.Claims, ids []int64) (Route, error) {
	var out Route
	err := s.store.InTx(ctx, func(q Store) error {
		r, err := s.ensure(ctx, q, driver)
		if err != nil {
			return err
		}
		items, err := q.ListRouteItems(ctx, r.ID)
		if err != nil {
			return err
		}
		current := make([]int64, len(items))
		for i, it := range items {
			current[i] = it.Delivery.ID
		}
		if !samePackages(current, ids) {
			return &apperr.ValidationError{Fields: map[string]string{
				"delivery_ids": "must list every package of the route once",
			}}
		}
		if err := q.SetRoutePositions(ctx, store.SetRoutePositionsParams{RouteID: r.ID, DeliveryIds: ids}); err != nil {
			return err
		}
		out, err = s.load(ctx, q, r)
		return err
	})
	return out, err
}

func samePackages(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// Optimize puts the stops in a short order, starting near start (the
// driver's position) when it is given. Stops without a map position go to
// the end, in the order they were.
func (s *Service) Optimize(ctx context.Context, driver auth.Claims, start *Point) (Route, error) {
	var out Route
	err := s.store.InTx(ctx, func(q Store) error {
		r, err := s.ensure(ctx, q, driver)
		if err != nil {
			return err
		}
		items, err := q.ListRouteItems(ctx, r.ID)
		if err != nil {
			return err
		}
		var onMap, offMap []Stop
		for _, st := range groupStops(items) {
			if st.location != nil {
				onMap = append(onMap, st)
			} else {
				offMap = append(offMap, st)
			}
		}
		points := make([]Point, len(onMap))
		for i, st := range onMap {
			points[i] = *st.location
		}
		var ids []int64
		for _, i := range order(points, start) {
			ids = append(ids, onMap[i].items...)
		}
		for _, st := range offMap {
			ids = append(ids, st.items...)
		}
		if err := q.SetRoutePositions(ctx, store.SetRoutePositionsParams{RouteID: r.ID, DeliveryIds: ids}); err != nil {
			return err
		}
		out, err = s.load(ctx, q, r)
		return err
	})
	return out, err
}

func (s *Service) ensure(ctx context.Context, q Store, driver auth.Claims) (store.Route, error) {
	return q.EnsureRoute(ctx, store.EnsureRouteParams{
		CarrierID: driver.CarrierID,
		DriverID:  driver.UserID,
		RouteDate: pgtype.Date{Time: s.today(), Valid: true},
	})
}

func (s *Service) today() time.Time {
	y, m, d := s.now().In(brazil).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *Service) load(ctx context.Context, q Store, r store.Route) (Route, error) {
	items, err := q.ListRouteItems(ctx, r.ID)
	if err != nil {
		return Route{}, err
	}
	out := Route{Date: r.RouteDate.Time.Format(time.DateOnly), Stops: groupStops(items)}
	for i := range out.Stops {
		for j := range out.Stops[i].Packages {
			out.TotalPackages++
			out.Stops[i].Packages[j].Position = out.TotalPackages
		}
	}
	return out, nil
}

// groupStops puts packages to the same address in one stop, at the place of
// the first of them, and numbers the stops. Items come in route order.
func groupStops(items []store.ListRouteItemsRow) []Stop {
	stops := []Stop{}
	index := map[string]int{}
	for _, it := range items {
		d := it.Delivery
		key := stopKey(d)
		i, ok := index[key]
		if !ok {
			i = len(stops)
			index[key] = i
			stops = append(stops, Stop{Number: i + 1, Address: stopAddress(d), Packages: []Package{}})
		}
		st := &stops[i]
		if st.location == nil && d.Latitude != nil && d.Longitude != nil {
			st.location = &Point{Lat: *d.Latitude, Lng: *d.Longitude}
			st.Latitude, st.Longitude = d.Latitude, d.Longitude
		}
		st.Packages = append(st.Packages, Package{Delivery: delivery.Delivery(d)})
		st.items = append(st.items, d.ID)
	}
	return stops
}

// stopAddress leaves out the complement, which can differ between the
// packages of a stop.
func stopAddress(d store.Delivery) string {
	if d.PostalCode == "" {
		return d.Address
	}
	return d.Street + ", " + d.Number + " - " + d.District + ", " + d.City
}

// stopKey is the same for packages delivered at the same door: same CEP,
// street and number, whatever the complement (apartments of a building are
// one stop). Older deliveries only have the one-line address.
func stopKey(d store.Delivery) string {
	if d.PostalCode != "" {
		return strings.ToLower(d.PostalCode + "|" + d.Street + "|" + d.Number)
	}
	return strings.ToLower(strings.Join(strings.Fields(d.Address), " "))
}
