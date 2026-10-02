// Package push lets the recipient ask for Web Push notifications about a
// delivery from the public tracking page. The worker sends them (see
// notify.Pusher).
package push

import (
	"context"
	"crypto/ecdh"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// MaxPerDelivery caps the browsers following one delivery, so a leaked code
// cannot fill the table.
const MaxPerDelivery = 10

// pushHosts are the push services of the browsers. The worker POSTs to the
// endpoint the browser hands over, so any other host is refused: otherwise
// anyone with a tracking code could make the worker call internal addresses
// (SSRF).
var pushHosts = []string{
	"fcm.googleapis.com",         // Chrome, Edge on Android, Samsung
	".push.services.mozilla.com", // Firefox
	".notify.windows.com",        // Edge on Windows
	"web.push.apple.com",         // Safari, iPhone
	".push.apple.com",
}

// Finder resolves a public tracking code. *delivery.Service implements it.
type Finder interface {
	PublicDelivery(ctx context.Context, code string) (store.Delivery, error)
}

type Store interface {
	UpsertPushSubscription(ctx context.Context, arg store.UpsertPushSubscriptionParams) error
	CountPushSubscriptions(ctx context.Context, deliveryID int64) (int64, error)
	DeletePushSubscription(ctx context.Context, arg store.DeletePushSubscriptionParams) error
}

// Subscription is what PushManager.subscribe() returns in the browser, as JSON.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type Service struct {
	finder Finder
	store  Store
}

func NewService(f Finder, s Store) *Service {
	return &Service{finder: f, store: s}
}

// Subscribe registers a browser for the delivery behind code.
func (s *Service) Subscribe(ctx context.Context, code string, in Subscription) error {
	v := apperr.Validator{}
	v.Check(validEndpoint(in.Endpoint), "endpoint", "must be an https URL of a browser push service")
	v.Check(validPoint(in.Keys.P256dh), "keys.p256dh", "must be a base64url P-256 public key")
	v.Check(validKey(in.Keys.Auth, 16), "keys.auth", "must be a base64url 16-byte secret")
	if err := v.Err(); err != nil {
		return err
	}
	d, err := s.finder.PublicDelivery(ctx, code)
	if err != nil {
		return err
	}
	if d.Status == "delivered" {
		return fmt.Errorf("%w: the delivery is already delivered", apperr.ErrConflict)
	}
	n, err := s.store.CountPushSubscriptions(ctx, d.ID)
	if err != nil {
		return err
	}
	if n >= MaxPerDelivery {
		return fmt.Errorf("%w: too many devices follow this delivery", apperr.ErrConflict)
	}
	return s.store.UpsertPushSubscription(ctx, store.UpsertPushSubscriptionParams{
		DeliveryID: d.ID, Endpoint: in.Endpoint, P256dh: in.Keys.P256dh, Auth: in.Keys.Auth,
	})
}

// Unsubscribe removes a browser. Unknown endpoints succeed, so the call can
// be repeated.
func (s *Service) Unsubscribe(ctx context.Context, code, endpoint string) error {
	d, err := s.finder.PublicDelivery(ctx, code)
	if err != nil {
		return err
	}
	return s.store.DeletePushSubscription(ctx, store.DeletePushSubscriptionParams{DeliveryID: d.ID, Endpoint: endpoint})
}

func validEndpoint(raw string) bool {
	if len(raw) > 1000 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range pushHosts {
		if host == h || (strings.HasPrefix(h, ".") && strings.HasSuffix(host, h)) {
			return true
		}
	}
	return false
}

// validPoint checks the browser key is a point on the curve; the worker
// could not encrypt for it otherwise and would retry forever.
func validPoint(s string) bool {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		return false
	}
	_, err = ecdh.P256().NewPublicKey(b)
	return err == nil
}

func validKey(s string, size int) bool {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	return err == nil && len(b) == size
}
