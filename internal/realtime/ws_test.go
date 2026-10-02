package realtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const origin = "http://front.test"

func newTestServer(t *testing.T, authorize Authorize, opts Options) (*Local, string, context.CancelFunc) {
	t.Helper()
	b := NewLocal()
	shutdown, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	opts.Origins = []string{origin}
	s := NewServer(shutdown, b, opts)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Stream(w, r, "topic", authorize)
	}))
	t.Cleanup(srv.Close)
	return b, "ws" + strings.TrimPrefix(srv.URL, "http"), stop
}

// dial connects from origin and returns the HTTP status of the handshake.
func dial(t *testing.T, url, from string) (*websocket.Conn, int, error) {
	t.Helper()
	c, res, err := websocket.Dial(t.Context(), url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {from}},
	})
	if c != nil {
		t.Cleanup(func() { _ = c.CloseNow() })
	}
	status := 0
	if res != nil {
		status = res.StatusCode
		if res.Body != nil {
			_ = res.Body.Close()
		}
	}
	return c, status, err
}

// waitSubscribed waits until n connections listen on the topic, so a publish
// is not lost to a race with the subscription.
func waitSubscribed(t *testing.T, b *Local, n int) {
	t.Helper()
	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return len(b.topics["topic"]) == n
	}, 2*time.Second, 5*time.Millisecond)
}

func read(t *testing.T, c *websocket.Conn) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	_, msg, err := c.Read(ctx)
	return string(msg), err
}

func TestStream_SendsPublishedMessages(t *testing.T) {
	b, url, _ := newTestServer(t, nil, Options{})
	c, _, err := dial(t, url, origin)
	require.NoError(t, err)
	waitSubscribed(t, b, 1)

	require.NoError(t, b.Publish(t.Context(), "topic", []byte(`{"status":"picked_up"}`)))

	msg, err := read(t, c)
	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"picked_up"}`, msg)
}

func TestStream_RejectsUnknownOrigin(t *testing.T) {
	_, url, _ := newTestServer(t, nil, Options{})
	_, status, err := dial(t, url, "http://evil.test")
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestStream_LimitsConnectionsPerIP(t *testing.T) {
	b, url, _ := newTestServer(t, nil, Options{MaxPerIP: 2})
	for range 2 {
		_, _, err := dial(t, url, origin)
		require.NoError(t, err)
	}
	waitSubscribed(t, b, 2)

	_, status, err := dial(t, url, origin)
	require.Error(t, err)
	assert.Equal(t, http.StatusTooManyRequests, status)
}

func TestStream_ClosesOnShutdown(t *testing.T) {
	b, url, shutdown := newTestServer(t, nil, Options{})
	c, _, err := dial(t, url, origin)
	require.NoError(t, err)
	waitSubscribed(t, b, 1)

	shutdown()
	_, err = read(t, c)
	assert.Equal(t, websocket.StatusGoingAway, websocket.CloseStatus(err))
}

func allowToken(valid string, ttl time.Duration) Authorize {
	return func(token string) (time.Time, error) {
		if token != valid {
			return time.Time{}, errors.New("bad token")
		}
		return time.Now().Add(ttl), nil
	}
}

func TestStream_Auth(t *testing.T) {
	t.Run("valid token gets messages", func(t *testing.T) {
		b, url, _ := newTestServer(t, allowToken("good", time.Minute), Options{})
		c, _, err := dial(t, url, origin)
		require.NoError(t, err)
		require.NoError(t, c.Write(t.Context(), websocket.MessageText, []byte(`{"token":"good"}`)))
		waitSubscribed(t, b, 1)

		require.NoError(t, b.Publish(t.Context(), "topic", []byte(`{}`)))
		_, err = read(t, c)
		require.NoError(t, err)
	})

	t.Run("bad token is closed as unauthorized", func(t *testing.T) {
		_, url, _ := newTestServer(t, allowToken("good", time.Minute), Options{})
		c, _, err := dial(t, url, origin)
		require.NoError(t, err)
		require.NoError(t, c.Write(t.Context(), websocket.MessageText, []byte(`{"token":"bad"}`)))

		_, err = read(t, c)
		assert.Equal(t, StatusUnauthorized, websocket.CloseStatus(err))
	})

	t.Run("expired token is closed as unauthorized", func(t *testing.T) {
		_, url, _ := newTestServer(t, allowToken("good", 50*time.Millisecond), Options{})
		c, _, err := dial(t, url, origin)
		require.NoError(t, err)
		require.NoError(t, c.Write(t.Context(), websocket.MessageText, []byte(`{"token":"good"}`)))

		_, err = read(t, c)
		assert.Equal(t, StatusUnauthorized, websocket.CloseStatus(err))
	})
}
