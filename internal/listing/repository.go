package listing

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a listing does not exist.
var ErrNotFound = errors.New("listing not found")

// ListFilter carries optional filters and pagination for List. Nil pointer
// fields mean "no filter on this field".
type ListFilter struct {
	City     *string
	Status   *Status
	MinPrice *int64
	MaxPrice *int64

	Limit  int
	Offset int
}

// Repository is the persistence contract the service depends on. The interface
// lives here, with the consumer, so the domain never imports its storage
// implementation. The Postgres implementation is in the store subpackage.
type Repository interface {
	Create(ctx context.Context, l *Listing) error
	GetByID(ctx context.Context, id uuid.UUID) (*Listing, error)
	List(ctx context.Context, f ListFilter) ([]Listing, error)
	Update(ctx context.Context, l *Listing) error
	Delete(ctx context.Context, id uuid.UUID) error
}
