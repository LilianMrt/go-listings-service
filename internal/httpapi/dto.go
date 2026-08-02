package httpapi

import (
	"time"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// listingResponse is the JSON representation of a listing.
type listingResponse struct {
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

func toResponse(l *listing.Listing) listingResponse {
	return listingResponse{
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

// createRequest is the POST body.
type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Currency    string `json:"currency"`
	City        string `json:"city"`
	PostalCode  string `json:"postal_code"`
	SurfaceM2   int    `json:"surface_m2"`
	Rooms       int    `json:"rooms"`
	SellerID    string `json:"seller_id"`
}

// updateRequest is the PATCH body: every field is optional. A nil pointer means
// "leave unchanged".
type updateRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	PriceCents  *int64  `json:"price_cents"`
	Currency    *string `json:"currency"`
	City        *string `json:"city"`
	PostalCode  *string `json:"postal_code"`
	SurfaceM2   *int    `json:"surface_m2"`
	Rooms       *int    `json:"rooms"`
}

func (r updateRequest) toInput() listing.UpdateInput {
	return listing.UpdateInput{
		Title:       r.Title,
		Description: r.Description,
		PriceCents:  r.PriceCents,
		Currency:    r.Currency,
		City:        r.City,
		PostalCode:  r.PostalCode,
		SurfaceM2:   r.SurfaceM2,
		Rooms:       r.Rooms,
	}
}

// listResponse wraps a page of listings with its pagination echo.
type listResponse struct {
	Items  []listingResponse `json:"items"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Count  int               `json:"count"`
}
