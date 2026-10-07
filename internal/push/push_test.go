package push_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/push"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type fakeFinder struct {
	d   store.Delivery
	err error
}

func (f fakeFinder) PublicDelivery(context.Context, string) (store.Delivery, error) {
	return f.d, f.err
}

type fakeStore struct {
	count int64
	saved []store.UpsertPushSubscriptionParams
}

func (f *fakeStore) UpsertPushSubscription(_ context.Context, p store.UpsertPushSubscriptionParams) error {
	f.saved = append(f.saved, p)
	return nil
}

func (f *fakeStore) CountPushSubscriptions(context.Context, store.CountPushSubscriptionsParams) (int64, error) {
	return f.count, nil
}

func (f *fakeStore) DeletePushSubscription(context.Context, store.DeletePushSubscriptionParams) error {
	return nil
}

func sub(endpoint string) push.Subscription {
	var s push.Subscription
	s.Endpoint = endpoint
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	s.Keys.P256dh = base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes())
	s.Keys.Auth = base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	return s
}

func inTransit() fakeFinder {
	return fakeFinder{d: store.Delivery{ID: 4, Status: "in_transit"}}
}

func TestSubscribe(t *testing.T) {
	st := &fakeStore{}
	svc := push.NewService(inTransit(), st)

	require.NoError(t, svc.Subscribe(context.Background(), "RSABCDEFGH23", sub("https://fcm.googleapis.com/fcm/send/abc")))
	require.Len(t, st.saved, 1)
	assert.Equal(t, int64(4), st.saved[0].DeliveryID)
}

func TestSubscribe_AcceptsEveryBrowser(t *testing.T) {
	for _, e := range []string{
		"https://fcm.googleapis.com/fcm/send/abc",
		"https://updates.push.services.mozilla.com/wpush/v2/abc",
		"https://wns2-par02p.notify.windows.com/w/?token=abc",
		"https://web.push.apple.com/abc",
	} {
		svc := push.NewService(inTransit(), &fakeStore{})
		assert.NoError(t, svc.Subscribe(context.Background(), "RSABCDEFGH23", sub(e)), e)
	}
}

// The worker POSTs to the endpoint, so anything but a push service is SSRF.
func TestSubscribe_RefusesOtherEndpoints(t *testing.T) {
	for _, e := range []string{
		"http://fcm.googleapis.com/fcm/send/abc",
		"https://169.254.169.254/latest/meta-data",
		"https://localhost/x",
		"https://fcm.googleapis.com.evil.com/x",
		"https://evilpush.services.mozilla.com.evil.com/x",
		"https://fcm.googleapis.com:8443/x",
		"https://user@fcm.googleapis.com/x",
		"https://fcm.googleapis.com/" + strings.Repeat("a", 1000),
		"",
	} {
		err := push.NewService(inTransit(), &fakeStore{}).Subscribe(context.Background(), "RSABCDEFGH23", sub(e))
		var verr *apperr.ValidationError
		require.ErrorAs(t, err, &verr, e)
		assert.Contains(t, verr.Fields, "endpoint", e)
	}
}

func TestSubscribe_ChecksKeys(t *testing.T) {
	s := sub("https://fcm.googleapis.com/fcm/send/abc")
	s.Keys.P256dh = base64.RawURLEncoding.EncodeToString(append([]byte{4}, make([]byte, 64)...)) // off the curve
	s.Keys.Auth = "!!"
	err := push.NewService(inTransit(), &fakeStore{}).Subscribe(context.Background(), "RSABCDEFGH23", s)
	var verr *apperr.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Fields, "keys.p256dh")
	assert.Contains(t, verr.Fields, "keys.auth")
}

func TestSubscribe_UnknownCode(t *testing.T) {
	svc := push.NewService(fakeFinder{err: apperr.ErrNotFound}, &fakeStore{})
	err := svc.Subscribe(context.Background(), "RSABCDEFGH23", sub("https://fcm.googleapis.com/fcm/send/abc"))
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestSubscribe_DeliveredOrFull(t *testing.T) {
	ok := sub("https://fcm.googleapis.com/fcm/send/abc")
	done := push.NewService(fakeFinder{d: store.Delivery{Status: "delivered"}}, &fakeStore{})
	assert.ErrorIs(t, done.Subscribe(context.Background(), "RSABCDEFGH23", ok), apperr.ErrConflict)

	full := push.NewService(inTransit(), &fakeStore{count: push.MaxPerDelivery})
	assert.ErrorIs(t, full.Subscribe(context.Background(), "RSABCDEFGH23", ok), apperr.ErrConflict)
}

// The front sends PushSubscription.toJSON() as the browser makes it,
// expirationTime included.
func TestHandler_SubscribeTakesTheBrowserJSON(t *testing.T) {
	st := &fakeStore{}
	r := chi.NewRouter()
	r.Post("/public/tracking/{code}/push", push.NewHandler(push.NewService(inTransit(), st), "key").Subscribe)

	s := sub("https://fcm.googleapis.com/fcm/send/abc")
	body := `{"endpoint":"` + s.Endpoint + `","expirationTime":null,"keys":{"p256dh":"` + s.Keys.P256dh + `","auth":"` + s.Keys.Auth + `"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/public/tracking/RSABCDEFGH23/push", strings.NewReader(body)))
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Len(t, st.saved, 1)
}
