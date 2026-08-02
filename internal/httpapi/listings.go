package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// maxBodyBytes caps request bodies to keep the service from buffering huge
// payloads.
const maxBodyBytes = 1 << 20 // 1 MiB

type listingHandler struct {
	svc    *listing.Service
	logger *slog.Logger
}

// routes returns the sub-router for /v1/listings.
func (h *listingHandler) routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	r.Post("/{id}/publish", h.publish)
	r.Post("/{id}/sell", h.sell)
	return r
}

func (h *listingHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	sellerID, err := uuid.Parse(req.SellerID)
	if err != nil {
		writeErrEnvelope(w, http.StatusUnprocessableEntity, "validation_failed",
			"one or more fields are invalid", map[string]string{"seller_id": "must be a valid uuid"})
		return
	}

	l, err := h.svc.Create(r.Context(), listing.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Currency:    req.Currency,
		City:        req.City,
		PostalCode:  req.PostalCode,
		SurfaceM2:   req.SurfaceM2,
		Rooms:       req.Rooms,
		SellerID:    sellerID,
	})
	if err != nil {
		writeServiceError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(l))
}

func (h *listingHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	l, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(l))
}

func (h *listingHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := listing.ListFilter{
		Limit:  20,
		Offset: 0,
	}

	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			writeBadRequest(w, "limit must be an integer between 1 and 100")
			return
		}
		filter.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeBadRequest(w, "offset must be a non-negative integer")
			return
		}
		filter.Offset = n
	}
	if v := q.Get("city"); v != "" {
		filter.City = &v
	}
	if v := q.Get("status"); v != "" {
		st := listing.Status(v)
		if !st.Valid() {
			writeBadRequest(w, "status must be one of draft, published, sold")
			return
		}
		filter.Status = &st
	}
	if v := q.Get("min_price"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			writeBadRequest(w, "min_price must be a non-negative integer")
			return
		}
		filter.MinPrice = &n
	}
	if v := q.Get("max_price"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			writeBadRequest(w, "max_price must be a non-negative integer")
			return
		}
		filter.MaxPrice = &n
	}

	items, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeServiceError(w, h.logger, r, err)
		return
	}

	resp := listResponse{
		Items:  make([]listingResponse, 0, len(items)),
		Limit:  filter.Limit,
		Offset: filter.Offset,
		Count:  len(items),
	}
	for i := range items {
		resp.Items = append(resp.Items, toResponse(&items[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *listingHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req updateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	l, err := h.svc.Update(r.Context(), id, req.toInput())
	if err != nil {
		writeServiceError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(l))
}

func (h *listingHandler) publish(w http.ResponseWriter, r *http.Request) {
	h.doTransition(w, r, h.svc.Publish)
}

func (h *listingHandler) sell(w http.ResponseWriter, r *http.Request) {
	h.doTransition(w, r, h.svc.Sell)
}

func (h *listingHandler) doTransition(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, id uuid.UUID) (*listing.Listing, error)) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	l, err := fn(r.Context(), id)
	if err != nil {
		writeServiceError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(l))
}

func (h *listingHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeServiceError(w, h.logger, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseID reads and validates the {id} path parameter. It writes a 400 and
// returns ok=false when the id is not a valid uuid.
func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeBadRequest(w, "id must be a valid uuid")
		return uuid.Nil, false
	}
	return id, true
}

// decodeJSON strictly decodes a single JSON object into dst, rejecting unknown
// fields and trailing content, with a body-size cap.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
