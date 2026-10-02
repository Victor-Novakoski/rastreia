package notify_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/notify"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type fakePushStore struct {
	subs        []store.PushSubscription
	deleted     []int64
	deletedCode string
}

func (f *fakePushStore) ListPushSubscriptionsByCode(context.Context, string) ([]store.PushSubscription, error) {
	return f.subs, nil
}

func (f *fakePushStore) DeletePushSubscriptionByID(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakePushStore) DeletePushSubscriptionsByCode(_ context.Context, code string) error {
	f.deletedCode = code
	return nil
}

// sender answers with the status of each subscription id.
func sender(status map[int64]int, got *[]notify.PushPayload) notify.PushSender {
	return func(_ context.Context, payload []byte, s store.PushSubscription) (int, error) {
		var p notify.PushPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return 0, err
		}
		*got = append(*got, p)
		if st, ok := status[s.ID]; ok {
			return st, nil
		}
		return http.StatusCreated, nil
	}
}

func subs(ids ...int64) []store.PushSubscription {
	out := make([]store.PushSubscription, len(ids))
	for i, id := range ids {
		out[i] = store.PushSubscription{ID: id, Endpoint: "https://fcm.googleapis.com/fcm/send/x"}
	}
	return out
}

func TestPush_SendsToEveryDevice(t *testing.T) {
	st := &fakePushStore{subs: subs(1, 2)}
	var got []notify.PushPayload
	c := notify.PushConsumer(st, sender(nil, &got), "https://rastreia.example.com/rastreio")

	require.NoError(t, c.Handle(context.Background(), msg("in_transit")))
	require.Len(t, got, 2)
	assert.Equal(t, notify.PushPayload{
		Title: "Sua entrega RSABCDEFGH23 está em rota",
		Body:  "Sua entrega saiu para entrega e chega em breve.",
		URL:   "https://rastreia.example.com/rastreio/RSABCDEFGH23",
		Tag:   "RSABCDEFGH23",
	}, got[0])
	assert.Empty(t, st.deleted)
	assert.Empty(t, st.deletedCode)
}

func TestPush_DeletesExpiredSubscriptions(t *testing.T) {
	st := &fakePushStore{subs: subs(1, 2, 3)}
	var got []notify.PushPayload
	c := notify.PushConsumer(st, sender(map[int64]int{1: http.StatusGone, 3: http.StatusNotFound}, &got), "http://x")

	require.NoError(t, c.Handle(context.Background(), msg("picked_up")))
	assert.Equal(t, []int64{1, 3}, st.deleted)
}

func TestPush_RetriesWhenThePushServiceFails(t *testing.T) {
	st := &fakePushStore{subs: subs(1)}
	var got []notify.PushPayload
	c := notify.PushConsumer(st, sender(map[int64]int{1: http.StatusServiceUnavailable}, &got), "http://x")

	assert.Error(t, c.Handle(context.Background(), msg("delivered")))
	assert.Empty(t, st.deletedCode, "kept for the retry")
}

func TestPush_ForgetsDevicesOnceDelivered(t *testing.T) {
	st := &fakePushStore{subs: subs(1)}
	var got []notify.PushPayload
	c := notify.PushConsumer(st, sender(nil, &got), "http://x")

	require.NoError(t, c.Handle(context.Background(), msg("delivered")))
	assert.Len(t, got, 1)
	assert.Equal(t, "RSABCDEFGH23", st.deletedCode)
}

func TestPush_UnknownStatusIsParked(t *testing.T) {
	c := notify.PushConsumer(&fakePushStore{}, nil, "http://x")
	var bad notify.ErrBadMessage
	assert.True(t, errors.As(c.Handle(context.Background(), msg("lost")), &bad))
}
