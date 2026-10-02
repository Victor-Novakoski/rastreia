// Package testdb gives integration tests a real, migrated PostgreSQL. One
// container starts per test binary and each test gets its own database.
// Tests are skipped with -short or when Docker is not available.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/Victor-Novakoski/rastreia/internal/database"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

var (
	once    sync.Once
	baseURL string
	initErr error
	counter atomic.Int64
)

// New returns a pool connected to an empty database with every migration applied.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: skipped with -short")
	}
	once.Do(start)
	if initErr != nil {
		t.Skipf("integration test: Docker not available: %v", initErr)
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := fmt.Sprintf("test_%d", counter.Add(1))
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	if err := database.Migrate(u.String()); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func start() {
	defer func() {
		// testcontainers panics in some setups without Docker.
		if r := recover(); r != nil {
			initErr = fmt.Errorf("%v", r)
		}
	}()
	ctx := context.Background()
	c, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("rastreia"),
		postgres.WithUsername("rastreia"),
		postgres.WithPassword("rastreia"),
		postgres.BasicWaitStrategies(),
		testcontainers.WithLogger(nopLogger{}),
	)
	if err != nil {
		initErr = err
		return
	}
	baseURL, initErr = c.ConnectionString(ctx, "sslmode=disable")
	if initErr == nil && !strings.HasPrefix(baseURL, "postgres") {
		initErr = fmt.Errorf("unexpected connection string %q", baseURL)
	}
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

// Carrier creates a carrier and returns its id, for tests that need users.
func Carrier(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	c, err := store.New(pool).CreateCarrier(context.Background(), store.CreateCarrierParams{Name: "Transportadora Teste"})
	if err != nil {
		t.Fatal(err)
	}
	return c.ID
}
