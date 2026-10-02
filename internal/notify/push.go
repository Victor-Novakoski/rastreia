package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// PushStore is what the push consumer needs from the database.
type PushStore interface {
	ListPushSubscriptionsByCode(ctx context.Context, code string) ([]store.PushSubscription, error)
	DeletePushSubscriptionByID(ctx context.Context, id int64) error
	DeletePushSubscriptionsByCode(ctx context.Context, code string) error
}

// PushSender posts one encrypted message to a browser's push service and
// returns the HTTP status it answered.
type PushSender func(ctx context.Context, payload []byte, s store.PushSubscription) (int, error)

// VAPID identifies this server to the push services.
type VAPID struct {
	PublicKey  string
	PrivateKey string
	// Subject is a mailto: or https: contact for the push services.
	Subject string
}

// WebPush sends with VAPID through the real push services.
func WebPush(v VAPID) PushSender {
	client := &http.Client{
		Timeout: 15 * time.Second,
		// The endpoint was checked against the push services when saved;
		// a redirect could still lead somewhere else.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return func(ctx context.Context, payload []byte, s store.PushSubscription) (int, error) {
		res, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: s.Endpoint,
			Keys:     webpush.Keys{P256dh: s.P256dh, Auth: s.Auth},
		}, &webpush.Options{
			HTTPClient:      client,
			Subscriber:      v.Subject,
			VAPIDPublicKey:  v.PublicKey,
			VAPIDPrivateKey: v.PrivateKey,
			TTL:             3600,
			Urgency:         webpush.UrgencyNormal,
		})
		if err != nil {
			return 0, err
		}
		defer func() { _ = res.Body.Close() }()
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return res.StatusCode, nil
	}
}

// PushPayload is what the service worker (web/public/sw.js) receives.
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	// Tag makes a new notification replace the previous one of the delivery.
	Tag string `json:"tag"`
}

// PushConsumer notifies every browser following the delivery. Subscriptions
// the push service says are gone are deleted, and so are all of them once
// the delivery is delivered: there is nothing left to tell.
func PushConsumer(st PushStore, send PushSender, trackingURL string) Consumer {
	return Consumer{Queue: PushQueue, Handle: func(ctx context.Context, m StatusChanged) error {
		c, ok := statusCopy[m.Status]
		if !ok {
			return ErrBadMessage{fmt.Errorf("unknown status %q", m.Status)}
		}
		payload, err := json.Marshal(PushPayload{
			Title: fmt.Sprintf(c.subject, m.TrackingCode),
			Body:  c.line,
			URL:   strings.TrimRight(trackingURL, "/") + "/" + url.PathEscape(m.TrackingCode),
			Tag:   m.TrackingCode,
		})
		if err != nil {
			return ErrBadMessage{err}
		}
		subs, err := st.ListPushSubscriptionsByCode(ctx, m.TrackingCode)
		if err != nil {
			return err
		}
		var (
			errs []error
			sent int
		)
		for _, s := range subs {
			status, err := send(ctx, payload, s)
			switch {
			case err == nil && status < 300:
				sent++
			case err != nil:
				errs = append(errs, err)
			case status == http.StatusNotFound || status == http.StatusGone:
				if err := st.DeletePushSubscriptionByID(ctx, s.ID); err != nil {
					errs = append(errs, err)
				}
			case status >= 500 || status == http.StatusTooManyRequests:
				errs = append(errs, fmt.Errorf("push service answered %d", status))
			case status >= 400:
				// Bad key or payload: retrying will not help.
				slog.Warn("push refused", "subscription_id", s.ID, "status", status)
			}
		}
		if err := errors.Join(errs...); err != nil {
			// A retry sends again to the ones that worked: at least once.
			return err
		}
		if m.Status == "delivered" {
			if err := st.DeletePushSubscriptionsByCode(ctx, m.TrackingCode); err != nil {
				return err
			}
		}
		if sent > 0 {
			slog.Info("push sent", "event_id", m.EventID, "status", m.Status, "devices", sent)
		}
		return nil
	}}
}
