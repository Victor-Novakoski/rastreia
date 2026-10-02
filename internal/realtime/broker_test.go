package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocal_DeliversToSubscribersOfTheTopic(t *testing.T) {
	b := NewLocal()
	ctx := t.Context()
	a, err := b.Subscribe(ctx, "a")
	require.NoError(t, err)
	other, err := b.Subscribe(ctx, "b")
	require.NoError(t, err)

	require.NoError(t, b.Publish(ctx, "a", []byte("hi")))

	assert.Equal(t, []byte("hi"), <-a)
	assert.Empty(t, other)
}

func TestLocal_ClosesWhenContextEnds(t *testing.T) {
	b := NewLocal()
	ctx, cancel := context.WithCancel(t.Context())
	ch, err := b.Subscribe(ctx, "a")
	require.NoError(t, err)

	cancel()
	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("channel not closed")
	}
	assert.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return len(b.topics) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestLocal_DropsSlowSubscriber(t *testing.T) {
	b := NewLocal()
	ctx := t.Context()
	ch, err := b.Subscribe(ctx, "a")
	require.NoError(t, err)

	for range subscriberBuffer + 1 {
		require.NoError(t, b.Publish(ctx, "a", []byte("x")))
	}
	n := 0
	for range ch {
		n++
	}
	assert.Equal(t, subscriberBuffer, n, "buffered messages arrive, then the channel closes")
}
