package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisGuard is a Guard whose counts live in Redis, shared by every API
// instance. Keys carry a hash of the e-mail, never the e-mail itself.
type RedisGuard struct {
	client   redis.UniversalClient
	baseLock time.Duration
	maxLock  time.Duration
}

func NewRedisGuard(client redis.UniversalClient) *RedisGuard {
	return &RedisGuard{client: client, baseLock: baseLock, maxLock: maxLock}
}

// failScript counts a failure and sets the lock in one step, so two
// instances failing at once cannot lose a count. The count expires maxLock
// after the last failure, so failures far apart start over.
var failScript = redis.NewScript(`
local fails = redis.call('INCR', KEYS[1])
redis.call('PEXPIRE', KEYS[1], ARGV[3])
local maxFails = tonumber(ARGV[1])
if fails >= maxFails then
  local lock = math.min(tonumber(ARGV[2]) * 2 ^ (fails - maxFails), tonumber(ARGV[3]))
  redis.call('SET', KEYS[2], '1', 'PX', math.floor(lock))
end
return fails
`)

func (g *RedisGuard) Check(ctx context.Context, email string) (time.Duration, error) {
	_, lock := g.keys(email)
	ttl, err := g.client.PTTL(ctx, lock).Result()
	if err != nil {
		return 0, err
	}
	return max(ttl, 0), nil
}

func (g *RedisGuard) Fail(ctx context.Context, email string) error {
	fails, lock := g.keys(email)
	return failScript.Run(ctx, g.client, []string{fails, lock},
		maxFails, g.baseLock.Milliseconds(), g.maxLock.Milliseconds()).Err()
}

func (g *RedisGuard) Success(ctx context.Context, email string) error {
	fails, lock := g.keys(email)
	return g.client.Del(ctx, fails, lock).Err()
}

func (g *RedisGuard) keys(email string) (fails, lock string) {
	sum := sha256.Sum256([]byte(key(email)))
	id := hex.EncodeToString(sum[:])
	return "rastreia:login:" + id + ":fails", "rastreia:login:" + id + ":lock"
}
