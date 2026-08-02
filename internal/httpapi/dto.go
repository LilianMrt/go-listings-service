package httpapi

import (
	"time"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// listingResponse is the JSON representation of a listing. Struct tags feed both
// the response body and the generated OpenAPI schema.
type listingResponse struct {
	ID          string    `json:"id" format:"uuid" doc:"Listing identifier"`
	Title       string    `json:"title" doc:"Listing title"`
	Description string    `json:"description" doc:"Free-text description"`
	PriceCents  int64     `json:"price_cents" doc:"Price in cents"`
	Currency    string    `json:"currency" doc:"ISO 4217 currency code"`
	City        string    `json:"city"`
	PostalCode  string    `json:"postal_code"`
	SurfaceM2   int       `json:"surface_m2" doc:"Surface area in square meters"`
	Rooms       int       `json:"rooms"`
	Status      string    `json:"status" enum:"draft,published,sold" doc:"Lifecycle status"`
	SellerID    string    `json:"seller_id" format:"uuid"`
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

// ListingOutput wraps a single listing response body.
type ListingOutput struct {
	Body listingResponse
}

// EmptyOutput carries no body (used for 204 responses).
type EmptyOutput struct{}

// --- Inputs ---

// CreateListingInput is the POST body. Fields without omitempty are required in
// the generated schema.
type CreateListingInput struct {
	Body struct {
		Title       string `json:"title" minLength:"1" maxLength:"200" example:"Sunny 2-room apartment" doc:"Listing title"`
		Description string `json:"description,omitempty" doc:"Free-text description"`
		PriceCents  int64  `json:"price_cents" minimum:"1" example:"250000" doc:"Price in cents"`
		Currency    string `json:"currency,omitempty" minLength:"3" maxLength:"3" doc:"ISO 4217 code; defaults to EUR"`
		City        string `json:"city" minLength:"1" example:"Toulouse"`
		PostalCode  string `json:"postal_code" minLength:"1" example:"31000"`
		SurfaceM2   int    `json:"surface_m2,omitempty" minimum:"0"`
		Rooms       int    `json:"rooms,omitempty" minimum:"0"`
		SellerID    string `json:"seller_id" format:"uuid" doc:"Owner/seller identifier"`
	}
}

// UpdateListingInput is a partial update: every body field is optional.
type UpdateListingInput struct {
	ID   string `path:"id" format:"uuid" doc:"Listing identifier"`
	Body struct {
		Title       *string `json:"title,omitempty" minLength:"1" maxLength:"200"`
		Description *string `json:"description,omitempty"`
		PriceCents  *int64  `json:"price_cents,omitempty" minimum:"1"`
		Currency    *string `json:"currency,omitempty" minLength:"3" maxLength:"3"`
		City        *string `json:"city,omitempty" minLength:"1"`
		PostalCode  *string `json:"postal_code,omitempty" minLength:"1"`
		SurfaceM2   *int    `json:"surface_m2,omitempty" minimum:"0"`
		Rooms       *int    `json:"rooms,omitempty" minimum:"0"`
	}
}

func (in UpdateListingInput) toServiceInput() listing.UpdateInput {
	return listing.UpdateInput{
		Title:       in.Body.Title,
		Description: in.Body.Description,
		PriceCents:  in.Body.PriceCents,
		Currency:    in.Body.Currency,
		City:        in.Body.City,
		PostalCode:  in.Body.PostalCode,
		SurfaceM2:   in.Body.SurfaceM2,
		Rooms:       in.Body.Rooms,
	}
}

// ListingIDInput carries just the {id} path parameter.
type ListingIDInput struct {
	ID string `path:"id" format:"uuid" doc:"Listing identifier"`
}

// ListListingsInput carries pagination and filter query parameters. Query
// params cannot be pointers in Huma, so absent numeric filters arrive as 0 and
// are treated as "unset".
type ListListingsInput struct {
	Limit    int    `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Page size"`
	Offset   int    `query:"offset" minimum:"0" default:"0" doc:"Rows to skip"`
	City     string `query:"city" doc:"Filter by city"`
	Status   string `query:"status" enum:"draft,published,sold" doc:"Filter by status"`
	MinPrice int64  `query:"min_price" minimum:"0" doc:"Minimum price in cents (0 = no minimum)"`
	MaxPrice int64  `query:"max_price" minimum:"0" doc:"Maximum price in cents (0 = no maximum)"`
}

func (in ListListingsInput) toFilter() listing.ListFilter {
	f := listing.ListFilter{
		Limit:  in.Limit,
		Offset: in.Offset,
	}
	if in.City != "" {
		f.City = &in.City
	}
	if in.Status != "" {
		s := listing.Status(in.Status)
		f.Status = &s
	}
	if in.MinPrice > 0 {
		v := in.MinPrice
		f.MinPrice = &v
	}
	if in.MaxPrice > 0 {
		v := in.MaxPrice
		f.MaxPrice = &v
	}
	return f
}

// ListListingsOutput is the paginated list response.
type ListListingsOutput struct {
	Body struct {
		Items  []listingResponse `json:"items"`
		Limit  int               `json:"limit"`
		Offset int               `json:"offset"`
		Count  int               `json:"count"`
	}
}
