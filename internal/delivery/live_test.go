package delivery

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type published struct {
	topic string
	msg   []byte
}

type fakePublisher struct {
	mu   sync.Mutex
	msgs []published
}

func (p *fakePublisher) Publish(_ context.Context, topic string, msg []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.msgs = append(p.msgs, published{topic, msg})
	return nil
}

func (p *fakePublisher) take() []published {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.msgs
	p.msgs = nil
	return out
}

func TestAnnounce_AddEventReachesPanelAndTracking(t *testing.T) {
	pub := &fakePublisher{}
	svc := NewService(newFakeStore()).WithPublisher(pub)
	d := newAssigned(t, svc)

	created := pub.take()
	require.Len(t, created, 1, "a new delivery only reaches the panel")
	assert.Equal(t, PanelTopic(1), created[0].topic)
	assert.JSONEq(t, `{"delivery_id":`+strconv.FormatInt(d.ID, 10)+`,"status":"pending"}`, string(created[0].msg))

	_, err := svc.AddEvent(context.Background(), driverA, d.ID, EventInput{Status: StatusPickedUp})
	require.NoError(t, err)

	msgs := pub.take()
	require.Len(t, msgs, 2)
	assert.Equal(t, PanelTopic(1), msgs[0].topic)
	assert.JSONEq(t, `{"delivery_id":`+strconv.FormatInt(d.ID, 10)+`,"status":"picked_up"}`, string(msgs[0].msg))

	assert.Equal(t, TrackingTopic(d.TrackingCode), msgs[1].topic)
	var tr Tracking
	require.NoError(t, json.Unmarshal(msgs[1].msg, &tr))
	assert.Equal(t, StatusPickedUp, tr.Status)
	assert.Len(t, tr.Events, 2)
	assert.NotContains(t, string(msgs[1].msg), "recipient_email", "the public stream keeps personal data out")
}

func TestAnnounce_RejectedChangePublishesNothing(t *testing.T) {
	pub := &fakePublisher{}
	svc := NewService(newFakeStore()).WithPublisher(pub)
	d := newAssigned(t, svc)
	pub.take()

	_, err := svc.AddEvent(context.Background(), driverA, d.ID, EventInput{Status: StatusDelivered})
	require.Error(t, err)
	assert.Empty(t, pub.take())
}

func TestAnnounce_RenameReloadsPublicPage(t *testing.T) {
	pub := &fakePublisher{}
	svc := NewService(newFakeStore()).WithPublisher(pub)
	d := newAssigned(t, svc)
	pub.take()

	_, err := svc.Update(context.Background(), owner, d.ID, UpdateInput{Number: ptr("20")})
	require.NoError(t, err)
	assert.Len(t, pub.take(), 1, "an address change only reaches the panel")

	_, err = svc.Update(context.Background(), owner, d.ID, UpdateInput{RecipientName: ptr("Maria Souza")})
	require.NoError(t, err)
	msgs := pub.take()
	require.Len(t, msgs, 2)
	assert.Contains(t, string(msgs[1].msg), `"recipient_first_name":"Maria"`)
}
