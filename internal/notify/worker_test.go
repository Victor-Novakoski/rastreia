package notify_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"

	"github.com/Victor-Novakoski/rastreia/internal/notify"
)

var (
	rabbitOnce sync.Once
	rabbitURL  string
	rabbitErr  error
)

// rabbit returns the URL of a real RabbitMQ, shared by the tests in this
// package. Tests run one at a time and purge the queues first.
func rabbit(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: skipped with -short")
	}
	rabbitOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				rabbitErr = fmt.Errorf("%v", r)
			}
		}()
		c, err := rabbitmq.Run(context.Background(), "rabbitmq:4-alpine")
		if err != nil {
			rabbitErr = err
			return
		}
		rabbitURL, rabbitErr = c.AmqpURL(context.Background())
	})
	if rabbitErr != nil {
		t.Skipf("integration test: Docker not available: %v", rabbitErr)
	}
	return rabbitURL
}

type fakeMailer struct {
	mu   sync.Mutex
	sent []notify.Email
	err  error
}

func (f *fakeMailer) Send(_ context.Context, e notify.Email) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, e)
	return nil
}

func (f *fakeMailer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

func publish(t *testing.T, url string, m notify.StatusChanged) {
	t.Helper()
	pub := notify.NewPublisher(url)
	defer pub.Close()
	body, err := json.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, pub.Publish(context.Background(), "1", body))
}

func queueLen(t *testing.T, url, queue string) int {
	t.Helper()
	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	require.NoError(t, err)
	q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
	if err != nil {
		return -1
	}
	return q.Messages
}

func purge(t *testing.T, url string) {
	t.Helper()
	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	require.NoError(t, err)
	require.NoError(t, notify.Declare(ch))
	for _, q := range []string{notify.EmailQueue, notify.EmailRetryQueue, notify.EmailDeadQueue} {
		_, err := ch.QueuePurge(q, false)
		require.NoError(t, err)
	}
}

func startWorker(t *testing.T, w *notify.Worker) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
}

func TestWorker_SendsEmail(t *testing.T) {
	url := rabbit(t)
	purge(t, url)
	// Published before the worker runs: Declare in the publisher keeps it.
	publish(t, url, msg("delivered"))

	mailer := &fakeMailer{}
	startWorker(t, notify.NewWorker(url, mailer, "http://localhost:5173/rastreio"))

	require.Eventually(t, func() bool { return mailer.count() == 1 }, 10*time.Second, 50*time.Millisecond)
	assert.Equal(t, "Sua entrega RSABCDEFGH23 foi entregue", mailer.sent[0].Subject)
}

func TestWorker_ParksAfterMaxAttempts(t *testing.T) {
	url := rabbit(t)
	purge(t, url)
	w := notify.NewWorker(url, &fakeMailer{err: errors.New("smtp down")}, "http://localhost:5173/rastreio")
	w.MaxAttempts = 1
	startWorker(t, w)
	publish(t, url, msg("delivered"))

	require.Eventually(t, func() bool { return queueLen(t, url, notify.EmailDeadQueue) == 1 }, 10*time.Second, 100*time.Millisecond)
	assert.Equal(t, 0, queueLen(t, url, notify.EmailRetryQueue))
}

func TestWorker_RetriesSendFailure(t *testing.T) {
	url := rabbit(t)
	purge(t, url)
	startWorker(t, notify.NewWorker(url, &fakeMailer{err: errors.New("smtp down")}, "http://localhost:5173/rastreio"))
	publish(t, url, msg("delivered"))

	require.Eventually(t, func() bool { return queueLen(t, url, notify.EmailRetryQueue) == 1 }, 10*time.Second, 100*time.Millisecond)
	assert.Equal(t, 0, queueLen(t, url, notify.EmailDeadQueue))
}

func TestWorker_ParksBadMessage(t *testing.T) {
	url := rabbit(t)
	purge(t, url)
	mailer := &fakeMailer{}
	startWorker(t, notify.NewWorker(url, mailer, "http://localhost:5173/rastreio"))
	publish(t, url, msg("lost"))

	require.Eventually(t, func() bool { return queueLen(t, url, notify.EmailDeadQueue) == 1 }, 10*time.Second, 100*time.Millisecond)
	assert.Zero(t, mailer.count())
}
