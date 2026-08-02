// Package events publishes listing domain events to Kafka. It implements the
// listing.Publisher contract; the interface lives with its consumer in the
// listing package. A Noop implementation is provided for tests and for running
// with Kafka disabled.
package events

import (
	"time"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// envelope is the wire format of a message on the listings topic:
//
//	{ "type": "...", "id": "...", "occurred_at": "...", "data": {...} }
//
// It is defined here, decoupled from the HTTP DTO, so the event schema is an
// explicit contract that does not drift with the API representation.
type envelope struct {
	Type       string      `json:"type"`
	ID         string      `json:"id"`
	OccurredAt time.Time   `json:"occurred_at"`
	Data       listingData `json:"data"`
}

// listingData is the snake_case payload embedded in an event.
type listingData struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Currency    string    `json:"currency"`
	City        string    `json:"city"`
	PostalCode  string    `json:"postal_code"`
	SurfaceM2   int       `json:"surface_m2"`
	Rooms       int       `json:"rooms"`
	Status      string    `json:"status"`
	SellerID    string    `json:"seller_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toData(l *listing.Listing) listingData {
	return listingData{
		ID:          l.ID.String(),
		Title:       l.Title,
		Description: l.Description,
		PriceCents:  l.PriceCents,
		Currency:    l.Currency,
		City:        l.City,
		PostalCode:  l.PostalCode,
		SurfaceM2:   l.SurfaceM2,
		Rooms:       l.Rooms,
		Status:      string(l.Status),
		SellerID:    l.SellerID.String(),
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
