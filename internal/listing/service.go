package listing

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service holds the business logic for listings. It depends on the storage
// contract (and, from M4, an event publisher) via interfaces so it is
// unit-testable with fakes.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService builds a Service over the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// CreateInput is the validated input for creating a listing.
type CreateInput struct {
	Title       string
	Description string
	PriceCents  int64
	Currency    string
	City        string
	PostalCode  string
	SurfaceM2   int
	Rooms       int
	SellerID    uuid.UUID
}

// Create validates the input and persists a new draft listing.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Listing, error) {
	in.Currency = normalizeCurrency(in.Currency)
	if err := validateFields(in.Title, in.Currency, in.City, in.PostalCode, in.PriceCents, in.SurfaceM2, in.Rooms); err != nil {
		return nil, err
	}
	if in.SellerID == uuid.Nil {
		return nil, &ValidationError{Fields: map[string]string{"seller_id": "is required"}}
	}

	now := s.now()
	l := &Listing{
		ID:          uuid.New(),
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		PriceCents:  in.PriceCents,
		Currency:    in.Currency,
		City:        strings.TrimSpace(in.City),
		PostalCode:  strings.TrimSpace(in.PostalCode),
		SurfaceM2:   in.SurfaceM2,
		Rooms:       in.Rooms,
		Status:      StatusDraft,
		SellerID:    in.SellerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// Get returns a listing by id, or ErrNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Listing, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns listings matching the filter.
func (s *Service) List(ctx context.Context, f ListFilter) ([]Listing, error) {
	return s.repo.List(ctx, f)
}

// UpdateInput is a partial update: only non-nil fields are applied.
type UpdateInput struct {
	Title       *string
	Description *string
	PriceCents  *int64
	Currency    *string
	City        *string
	PostalCode  *string
	SurfaceM2   *int
	Rooms       *int
}

// Update applies a partial update, re-validates the resulting listing, and
// persists it. Status is not updatable here; use Publish/Sell.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*Listing, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		l.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		l.Description = *in.Description
	}
	if in.PriceCents != nil {
		l.PriceCents = *in.PriceCents
	}
	if in.Currency != nil {
		l.Currency = normalizeCurrency(*in.Currency)
	}
	if in.City != nil {
		l.City = strings.TrimSpace(*in.City)
	}
	if in.PostalCode != nil {
		l.PostalCode = strings.TrimSpace(*in.PostalCode)
	}
	if in.SurfaceM2 != nil {
		l.SurfaceM2 = *in.SurfaceM2
	}
	if in.Rooms != nil {
		l.Rooms = *in.Rooms
	}

	if err := validateFields(l.Title, l.Currency, l.City, l.PostalCode, l.PriceCents, l.SurfaceM2, l.Rooms); err != nil {
		return nil, err
	}

	l.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// Publish moves a listing to published.
func (s *Service) Publish(ctx context.Context, id uuid.UUID) (*Listing, error) {
	return s.transition(ctx, id, StatusPublished)
}

// Sell moves a listing to sold.
func (s *Service) Sell(ctx context.Context, id uuid.UUID) (*Listing, error) {
	return s.transition(ctx, id, StatusSold)
}

func (s *Service) transition(ctx context.Context, id uuid.UUID, to Status) (*Listing, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !l.Status.CanTransitionTo(to) {
		return nil, &TransitionError{From: l.Status, To: to}
	}
	l.Status = to
	l.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// Delete removes a listing, or returns ErrNotFound.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
