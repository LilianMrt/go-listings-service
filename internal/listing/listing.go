// Package listing holds the domain model, the repository contract it depends
// on, and (from M2) the business-logic service. The concrete Postgres
// repository lives in the store subpackage.
package listing

import (
	"time"

	"github.com/google/uuid"
)

// Status is the lifecycle state of a listing. It mirrors the listing_status
// enum in the database.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusSold      Status = "sold"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusPublished, StatusSold:
		return true
	default:
		return false
	}
}

// Listing is a property listing.
type Listing struct {
	ID          uuid.UUID
	Title       string
	Description string
	PriceCents  int64
	Currency    string
	City        string
	PostalCode  string
	SurfaceM2   int
	Rooms       int
	Status      Status
	SellerID    uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
