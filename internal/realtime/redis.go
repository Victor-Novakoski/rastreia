package realtime

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

// channelPrefix keeps our channels apart from anything else on the Redis.
const channelPrefix = "rastreia:live:"

// Redis is a Broker shared by every API instance: Publish goes through
// Redis, and each instance holds one pattern subscription that hands the
// messages to its own connections. That is one Redis connection per
// instance, not one per browser.
//
// Messages published while an instance is cut off from Redis are lost to
// its browsers; they catch up on their next load, as with any dropped
// connection.
type Redis struct {
	client redis.UniversalClient
	local  *Local
}

// NewRedis subscribes to Redis and stops when ctx ends.
func NewRedis(ctx context.Context, client redis.UniversalClient) (*Redis, error) {
	ps := client.PSubscribe(ctx, channelPrefix+"*")
	// Wait for the confirmation, so a publish right after this is not missed.
	if _, err := ps.Receive(ctx); err != nil {
		_ = ps.Close()
		return nil, err
	}
	b := &Redis{client: client, local: NewLocal()}
	go func() {
		defer func() { _ = ps.Close() }()
		ch := ps.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				_ = b.local.Publish(ctx, strings.TrimPrefix(msg.Channel, channelPrefix), []byte(msg.Payload))
			}
		}
	}()
	return b, nil
}

func (b *Redis) Publish(ctx context.Context, topic string, msg []byte) error {
	return b.client.Publish(ctx, channelPrefix+topic, msg).Err()
}

func (b *Redis) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	return b.local.Subscribe(ctx, topic)
}
