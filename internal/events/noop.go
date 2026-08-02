package events

import (
	"context"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// Noop is a Publisher that discards events. It is used when Kafka is disabled
// and in tests that do not care about event delivery.
type Noop struct{}

var _ listing.Publisher = Noop{}

// Publish discards the event and always succeeds.
func (Noop) Publish(context.Context, listing.EventType, *listing.Listing) error { return nil }
