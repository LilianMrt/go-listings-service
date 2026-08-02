package listing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// fakeRepo is an in-memory Repository for unit-testing the service.
type fakeRepo struct {
	items    map[uuid.UUID]listing.Listing
	createErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[uuid.UUID]listing.Listing{}}
}

func (f *fakeRepo) Create(_ context.Context, l *listing.Listing) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.items[l.ID] = *l
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (*listing.Listing, error) {
	l, ok := f.items[id]
	if !ok {
		return nil, listing.ErrNotFound
	}
	return &l, nil
}

func (f *fakeRepo) List(_ context.Context, _ listing.ListFilter) ([]listing.Listing, error) {
	out := make([]listing.Listing, 0, len(f.items))
	for _, l := range f.items {
		out = append(out, l)
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, l *listing.Listing) error {
	if _, ok := f.items[l.ID]; !ok {
		return listing.ErrNotFound
	}
	f.items[l.ID] = *l
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.items[id]; !ok {
		return listing.ErrNotFound
	}
	delete(f.items, id)
	return nil
}

func validCreateInput() listing.CreateInput {
	return listing.CreateInput{
		Title:      "Nice flat",
		PriceCents: 250000,
		City:       "Toulouse",
		PostalCode: "31000",
		SurfaceM2:  45,
		Rooms:      2,
		SellerID:   uuid.New(),
	}
}

func TestCreate_DefaultsAndDraft(t *testing.T) {
	svc := listing.NewService(newFakeRepo())

	l, err := svc.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if l.Status != listing.StatusDraft {
		t.Errorf("status = %q, want draft", l.Status)
	}
	if l.Currency != "EUR" {
		t.Errorf("currency = %q, want default EUR", l.Currency)
	}
	if l.ID == uuid.Nil {
		t.Error("id was not assigned")
	}
	if l.CreatedAt.IsZero() || l.UpdatedAt.IsZero() {
		t.Error("timestamps not set")
	}
}

func TestCreate_Validation(t *testing.T) {
	svc := listing.NewService(newFakeRepo())

	in := validCreateInput()
	in.Title = ""
	in.PriceCents = 0

	_, err := svc.Create(context.Background(), in)
	var ve *listing.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	if _, ok := ve.Fields["title"]; !ok {
		t.Error("expected title error")
	}
	if _, ok := ve.Fields["price_cents"]; !ok {
		t.Error("expected price_cents error")
	}
}

func TestTransitions(t *testing.T) {
	ctx := context.Background()

	t.Run("draft cannot be sold", func(t *testing.T) {
		svc := listing.NewService(newFakeRepo())
		l, _ := svc.Create(ctx, validCreateInput())

		_, err := svc.Sell(ctx, l.ID)
		var te *listing.TransitionError
		if !errors.As(err, &te) {
			t.Fatalf("want TransitionError, got %v", err)
		}
	})

	t.Run("draft to published to sold", func(t *testing.T) {
		svc := listing.NewService(newFakeRepo())
		l, _ := svc.Create(ctx, validCreateInput())

		pub, err := svc.Publish(ctx, l.ID)
		if err != nil {
			t.Fatalf("publish: %v", err)
		}
		if pub.Status != listing.StatusPublished {
			t.Fatalf("status = %q, want published", pub.Status)
		}

		sold, err := svc.Sell(ctx, l.ID)
		if err != nil {
			t.Fatalf("sell: %v", err)
		}
		if sold.Status != listing.StatusSold {
			t.Fatalf("status = %q, want sold", sold.Status)
		}
	})
}

func TestUpdate_PartialAndValidation(t *testing.T) {
	ctx := context.Background()
	svc := listing.NewService(newFakeRepo())
	l, _ := svc.Create(ctx, validCreateInput())

	newTitle := "Updated title"
	updated, err := svc.Update(ctx, l.ID, listing.UpdateInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("title = %q, want %q", updated.Title, newTitle)
	}
	if updated.City != "Toulouse" {
		t.Errorf("city changed unexpectedly to %q", updated.City)
	}

	badPrice := int64(0)
	_, err = svc.Update(ctx, l.ID, listing.UpdateInput{PriceCents: &badPrice})
	var ve *listing.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
}

func TestGetAndDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	svc := listing.NewService(newFakeRepo())

	if _, err := svc.Get(ctx, uuid.New()); !errors.Is(err, listing.ErrNotFound) {
		t.Errorf("get: want ErrNotFound, got %v", err)
	}
	if err := svc.Delete(ctx, uuid.New()); !errors.Is(err, listing.ErrNotFound) {
		t.Errorf("delete: want ErrNotFound, got %v", err)
	}
}
