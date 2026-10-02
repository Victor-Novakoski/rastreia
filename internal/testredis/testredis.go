// Package testredis gives integration tests a real Redis. One container
// starts per test binary and each test gets its own empty logical database.
// Tests are skipped with -short or when Docker is not available.
package testredis

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// Redis has 16 logical databases by default; tests rotate through them.
const databases = 16

var (
	once    sync.Once
	baseURL string
	initErr error
	counter atomic.Int64
)

// New returns a client on an empty database. Pub/sub is shared by all
// databases, so tests that publish should use their own channel names.
func New(t *testing.T) *redis.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: skipped with -short")
	}
	once.Do(start)
	if initErr != nil {
		t.Skipf("integration test: Docker not available: %v", initErr)
	}

	opts, err := redis.ParseURL(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	opts.DB = int(counter.Add(1) % databases)
	c := redis.NewClient(opts)
	t.Cleanup(func() { _ = c.Close() })
	if err := c.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	return c
}

func start() {
	defer func() {
		// testcontainers panics in some setups without Docker.
		if r := recover(); r != nil {
			initErr = fmt.Errorf("%v", r)
		}
	}()
	ctx := context.Background()
	c, err := tcredis.Run(ctx, "redis:8-alpine", testcontainers.WithLogger(nopLogger{}))
	if err != nil {
		initErr = err
		return
	}
	baseURL, initErr = c.ConnectionString(ctx)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}
