// Package realtime pushes changes to browsers over WebSocket.
package realtime

import (
	"context"
	"sync"
)

// Broker delivers each message published on a topic to everyone subscribed
// to it at that moment. Nothing is stored: a browser that reconnects loads
// the current state over HTTP.
type Broker interface {
	Publish(ctx context.Context, topic string, msg []byte) error
	// Subscribe returns a channel that is closed when ctx ends, or earlier
	// when the subscriber falls too far behind.
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)
}

// subscriberBuffer is how many messages a slow connection may have pending
// before it is dropped.
const subscriberBuffer = 16

// Local is a Broker that reaches only the connections of this process.
type Local struct {
	mu     sync.Mutex
	topics map[string]map[chan []byte]struct{}
}

func NewLocal() *Local {
	return &Local{topics: map[string]map[chan []byte]struct{}{}}
}

func (b *Local) Publish(_ context.Context, topic string, msg []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.topics[topic] {
		select {
		case ch <- msg:
		default:
			// Dropping the subscriber makes its browser reconnect and reload,
			// instead of silently missing an update.
			b.remove(topic, ch)
		}
	}
	return nil
}

func (b *Local) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	ch := make(chan []byte, subscriberBuffer)
	b.mu.Lock()
	if b.topics[topic] == nil {
		b.topics[topic] = map[chan []byte]struct{}{}
	}
	b.topics[topic][ch] = struct{}{}
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		b.remove(topic, ch)
		b.mu.Unlock()
	}()
	return ch, nil
}

// remove must be called with mu held. It is safe to call twice.
func (b *Local) remove(topic string, ch chan []byte) {
	subs := b.topics[topic]
	if _, ok := subs[ch]; !ok {
		return
	}
	delete(subs, ch)
	close(ch)
	if len(subs) == 0 {
		delete(b.topics, topic)
	}
}
