package listing

import "context"

// EventType identifies a domain event emitted when a listing changes. The
// values are the message types written to the Kafka topic.
type EventType string

const (
	EventCreated EventType = "listing.created"
	EventUpdated EventType = "listing.updated"
	EventDeleted EventType = "listing.deleted"
)

// Publisher publishes a domain event after a listing mutation has been
// committed. Like Repository, the interface lives with its consumer (the
// service); the Kafka implementation and a noop live in internal/events.
//
// Publishing is best-effort and happens after the database commit, so an error
// here does not undo the mutation (at-least-once, no outbox in v1).
type Publisher interface {
	Publish(ctx context.Context, typ EventType, l *Listing) error
}

// nopPublisher is the default when no Publisher is configured (unit tests, or
// Kafka disabled without an explicit noop). It discards events.
type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, EventType, *Listing) error { return nil }
