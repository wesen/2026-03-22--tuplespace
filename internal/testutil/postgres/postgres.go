package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/manuel/wesen/tuplespace/internal/migrations"
)

type TestDatabase struct {
	Container *tcpostgres.PostgresContainer
	Pool      *pgxpool.Pool
	URL       string
}

func Start(t *testing.T) *TestDatabase {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("tuplespace"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("create pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	migrationsFS := os.DirFS(filepath.Join(projectRoot, "migrations"))
	if err := migrations.ApplyFS(ctx, pool, migrationsFS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return &TestDatabase{
		Container: container,
		Pool:      pool,
		URL:       url,
	}
}

func CountTuples(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM tuples`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tuples: %w", err)
	}
	return count, nil
}
