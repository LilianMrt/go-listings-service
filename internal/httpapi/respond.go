package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// errorBody is the consistent error envelope returned by every endpoint.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeErrEnvelope(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message, Fields: fields}})
}

// writeBadRequest is for malformed input the service never sees (bad JSON,
// unparseable ids or query params).
func writeBadRequest(w http.ResponseWriter, message string) {
	writeErrEnvelope(w, http.StatusBadRequest, "bad_request", message, nil)
}

// writeServiceError maps domain errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, logger *slog.Logger, r *http.Request, err error) {
	var ve *listing.ValidationError
	var te *listing.TransitionError

	switch {
	case errors.As(err, &ve):
		writeErrEnvelope(w, http.StatusUnprocessableEntity, "validation_failed", "one or more fields are invalid", ve.Fields)
	case errors.As(err, &te):
		writeErrEnvelope(w, http.StatusConflict, "invalid_transition", te.Error(), nil)
	case errors.Is(err, listing.ErrNotFound):
		writeErrEnvelope(w, http.StatusNotFound, "not_found", "listing not found", nil)
	default:
		logger.ErrorContext(r.Context(), "unhandled service error", slog.Any("error", err))
		writeErrEnvelope(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}
