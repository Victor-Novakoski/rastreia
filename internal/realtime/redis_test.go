package realtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/testredis"
)

func TestRedis_ReachesOtherInstances(t *testing.T) {
	client := testredis.New(t)
	// Two brokers on the same Redis stand in for two API instances.
	a, err := NewRedis(t.Context(), client)
	require.NoError(t, err)
	b, err := NewRedis(t.Context(), client)
	require.NoError(t, err)

	msgs, err := b.Subscribe(t.Context(), "tracking:RS7K2M9QXA4P")
	require.NoError(t, err)
	other, err := b.Subscribe(t.Context(), "deliveries")
	require.NoError(t, err)

	require.NoError(t, a.Publish(t.Context(), "tracking:RS7K2M9QXA4P", []byte(`{"status":"delivered"}`)))

	select {
	case msg := <-msgs:
		assert.JSONEq(t, `{"status":"delivered"}`, string(msg))
	case <-time.After(2 * time.Second):
		t.Fatal("message did not cross instances")
	}
	assert.Empty(t, other)
}
