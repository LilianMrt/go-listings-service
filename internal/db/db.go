// Package db handles the database connection pool and schema migrations.
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LilianMrt/go-listings-service/migrations"
)

// NewPool opens a pgx connection pool and verifies connectivity.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

// Migrate applies all up migrations. It is a no-op if the schema is current.
//
// Tradeoff: migrating on startup keeps local dev and tests simple, but in a
// real multi-replica deployment you would run migrations as a separate step to
// avoid races. Documented in the README.
func Migrate(databaseURL string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, toMigrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// toMigrateURL rewrites the connection URL to the scheme golang-migrate's
// pgx/v5 driver registers ("pgx5"), leaving other schemes untouched.
func toMigrateURL(databaseURL string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(databaseURL, prefix) {
			return "pgx5://" + strings.TrimPrefix(databaseURL, prefix)
		}
	}
	return databaseURL
}
