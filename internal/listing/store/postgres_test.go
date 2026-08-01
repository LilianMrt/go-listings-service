package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/LilianMrt/go-listings-service/internal/db"
	"github.com/LilianMrt/go-listings-service/internal/listing"
	"github.com/LilianMrt/go-listings-service/internal/listing/store"
)

// TestPostgresCRUD exercises the repository against a real Postgres. It is
// skipped unless TEST_DATABASE_URL is set. M3 will drive this via
// testcontainers so it runs unattended.
func TestPostgresCRUD(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the store integration test")
	}

	if err := db.Migrate(url); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	repo := store.NewPostgres(pool)

	now := time.Now().UTC().Truncate(time.Microsecond)
	l := &listing.Listing{
		ID:          uuid.New(),
		Title:       "Nice flat",
		Description: "sunny 2-room",
		PriceCents:  250000,
		Currency:    "EUR",
		City:        "Toulouse",
		PostalCode:  "31000",
		SurfaceM2:   45,
		Rooms:       2,
		Status:      listing.StatusDraft,
		SellerID:    uuid.New(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := repo.Create(ctx, l); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(context.Background(), l.ID) })

	got, err := repo.GetByID(ctx, l.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != l.Title || got.Status != listing.StatusDraft || got.City != l.City {
		t.Fatalf("unexpected read: %+v", got)
	}

	got.Status = listing.StatusPublished
	got.PriceCents = 260000
	got.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	city := l.City
	published := listing.StatusPublished
	items, err := repo.List(ctx, listing.ListFilter{City: &city, Status: &published, Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, it := range items {
		if it.ID == l.ID {
			found = true
			if it.PriceCents != 260000 {
				t.Fatalf("update not persisted, price = %d", it.PriceCents)
			}
		}
	}
	if !found {
		t.Fatal("updated listing not returned by List")
	}

	if err := repo.Delete(ctx, l.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, l.ID); !errors.Is(err, listing.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
