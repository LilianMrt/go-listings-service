package httpapi

import (
	"context"
	"errors"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// toHumaError maps domain errors onto Huma's RFC 7807 problem+json responses.
// Unknown errors are logged and surfaced as a generic 500.
func toHumaError(ctx context.Context, logger *slog.Logger, err error) error {
	var ve *listing.ValidationError
	var te *listing.TransitionError

	switch {
	case errors.As(err, &ve):
		details := make([]error, 0, len(ve.Fields))
		for field, msg := range ve.Fields {
			details = append(details, &huma.ErrorDetail{
				Message:  msg,
				Location: "body." + field,
			})
		}
		return huma.Error422UnprocessableEntity("one or more fields are invalid", details...)
	case errors.As(err, &te):
		return huma.Error409Conflict(te.Error())
	case errors.Is(err, listing.ErrNotFound):
		return huma.Error404NotFound("listing not found")
	default:
		logger.ErrorContext(ctx, "unhandled service error", slog.Any("error", err))
		return huma.Error500InternalServerError("internal server error")
	}
}
