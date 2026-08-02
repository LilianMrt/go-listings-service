package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

type listingRoutes struct {
	svc    *listing.Service
	logger *slog.Logger
}

// registerListings wires the /v1/listings operations onto the Huma API. Each
// registration contributes to the generated OpenAPI document.
func registerListings(api huma.API, svc *listing.Service, logger *slog.Logger) {
	h := &listingRoutes{svc: svc, logger: logger}
	tag := []string{"Listings"}

	huma.Register(api, huma.Operation{
		OperationID:   "create-listing",
		Method:        http.MethodPost,
		Path:          "/v1/listings",
		Summary:       "Create a listing",
		Description:   "Creates a new listing in draft status.",
		Tags:          tag,
		DefaultStatus: http.StatusCreated,
	}, h.create)

	huma.Register(api, huma.Operation{
		OperationID: "list-listings",
		Method:      http.MethodGet,
		Path:        "/v1/listings",
		Summary:     "List listings",
		Description: "Lists listings with pagination and optional filters.",
		Tags:        tag,
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "get-listing",
		Method:      http.MethodGet,
		Path:        "/v1/listings/{id}",
		Summary:     "Get a listing",
		Tags:        tag,
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "update-listing",
		Method:      http.MethodPatch,
		Path:        "/v1/listings/{id}",
		Summary:     "Update a listing",
		Description: "Partially updates a listing. Only provided fields change.",
		Tags:        tag,
	}, h.update)

	huma.Register(api, huma.Operation{
		OperationID: "publish-listing",
		Method:      http.MethodPost,
		Path:        "/v1/listings/{id}/publish",
		Summary:     "Publish a listing",
		Description: "Transitions a draft listing to published.",
		Tags:        tag,
	}, h.publish)

	huma.Register(api, huma.Operation{
		OperationID: "sell-listing",
		Method:      http.MethodPost,
		Path:        "/v1/listings/{id}/sell",
		Summary:     "Mark a listing as sold",
		Description: "Transitions a published listing to sold.",
		Tags:        tag,
	}, h.sell)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-listing",
		Method:        http.MethodDelete,
		Path:          "/v1/listings/{id}",
		Summary:       "Delete a listing",
		Tags:          tag,
		DefaultStatus: http.StatusNoContent,
	}, h.delete)
}

func (h *listingRoutes) create(ctx context.Context, in *CreateListingInput) (*ListingOutput, error) {
	sellerID, err := uuid.Parse(in.Body.SellerID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("one or more fields are invalid",
			&huma.ErrorDetail{Message: "must be a valid uuid", Location: "body.seller_id", Value: in.Body.SellerID})
	}

	l, err := h.svc.Create(ctx, listing.CreateInput{
		Title:       in.Body.Title,
		Description: in.Body.Description,
		PriceCents:  in.Body.PriceCents,
		Currency:    in.Body.Currency,
		City:        in.Body.City,
		PostalCode:  in.Body.PostalCode,
		SurfaceM2:   in.Body.SurfaceM2,
		Rooms:       in.Body.Rooms,
		SellerID:    sellerID,
	})
	if err != nil {
		return nil, toHumaError(ctx, h.logger, err)
	}
	return &ListingOutput{Body: toResponse(l)}, nil
}

func (h *listingRoutes) get(ctx context.Context, in *ListingIDInput) (*ListingOutput, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("id must be a valid uuid")
	}
	l, err := h.svc.Get(ctx, id)
	if err != nil {
		return nil, toHumaError(ctx, h.logger, err)
	}
	return &ListingOutput{Body: toResponse(l)}, nil
}

func (h *listingRoutes) list(ctx context.Context, in *ListListingsInput) (*ListListingsOutput, error) {
	items, err := h.svc.List(ctx, in.toFilter())
	if err != nil {
		return nil, toHumaError(ctx, h.logger, err)
	}

	out := &ListListingsOutput{}
	out.Body.Items = make([]listingResponse, 0, len(items))
	for i := range items {
		out.Body.Items = append(out.Body.Items, toResponse(&items[i]))
	}
	out.Body.Limit = in.Limit
	out.Body.Offset = in.Offset
	out.Body.Count = len(items)
	return out, nil
}

func (h *listingRoutes) update(ctx context.Context, in *UpdateListingInput) (*ListingOutput, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("id must be a valid uuid")
	}
	l, err := h.svc.Update(ctx, id, in.toServiceInput())
	if err != nil {
		return nil, toHumaError(ctx, h.logger, err)
	}
	return &ListingOutput{Body: toResponse(l)}, nil
}

func (h *listingRoutes) publish(ctx context.Context, in *ListingIDInput) (*ListingOutput, error) {
	return h.transition(ctx, in.ID, h.svc.Publish)
}

func (h *listingRoutes) sell(ctx context.Context, in *ListingIDInput) (*ListingOutput, error) {
	return h.transition(ctx, in.ID, h.svc.Sell)
}

func (h *listingRoutes) transition(ctx context.Context, rawID string, fn func(context.Context, uuid.UUID) (*listing.Listing, error)) (*ListingOutput, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("id must be a valid uuid")
	}
	l, err := fn(ctx, id)
	if err != nil {
		return nil, toHumaError(ctx, h.logger, err)
	}
	return &ListingOutput{Body: toResponse(l)}, nil
}

func (h *listingRoutes) delete(ctx context.Context, in *ListingIDInput) (*EmptyOutput, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("id must be a valid uuid")
	}
	if err := h.svc.Delete(ctx, id); err != nil {
		return nil, toHumaError(ctx, h.logger, err)
	}
	return &EmptyOutput{}, nil
}
