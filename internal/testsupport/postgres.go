// Package testsupport provides shared helpers for integration tests. It is
// imported only from _test.go files, so its testcontainers dependency never
// ends up in the application binary.
package testsupport

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LilianMrt/go-listings-service/internal/db"
)

// StartPostgres launches a throwaway Postgres container, applies migrations,
// and returns its connection string. The container is terminated on cleanup.
func StartPostgres(t testing.TB) string {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("listings"),
		postgres.WithUsername("listings"),
		postgres.WithPassword("listings"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	if err := db.Migrate(dsn); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return dsn
}
