package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/LilianMrt/go-listings-service/internal/listing"
)

// writeAttempts bounds the retries on a cold-start topic error. With
// AllowAutoTopicCreation, the first produce to a new topic triggers creation
// but fails; a few short retries let creation settle so the event still lands.
const writeAttempts = 6

// Writer publishes listing events to a Kafka topic. Messages are keyed by the
// listing id, so all events for a given listing land on the same partition and
// stay ordered.
type Writer struct {
	w   *kafka.Writer
	now func() time.Time
}

var _ listing.Publisher = (*Writer)(nil)

// NewWriter builds a Kafka-backed publisher for the given brokers and topic.
// The topic is auto-created on first write; Publish retries briefly so that
// first event is not lost to the creation race. In production the topic is
// typically provisioned ahead of time.
func NewWriter(brokers []string, topic string) *Writer {
	return &Writer{
		w: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{}, // partition by message key (listing id)
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: true,
		},
		now: func() time.Time { return time.Now().UTC() },
	}
}

// Publish writes a single event to the topic, keyed by the listing id.
func (wr *Writer) Publish(ctx context.Context, typ listing.EventType, l *listing.Listing) error {
	payload, err := json.Marshal(envelope{
		Type:       string(typ),
		ID:         l.ID.String(),
		OccurredAt: wr.now(),
		Data:       toData(l),
	})
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	msg := kafka.Message{Key: []byte(l.ID.String()), Value: payload}

	for attempt := 0; attempt < writeAttempts; attempt++ {
		if err = wr.w.WriteMessages(ctx, msg); err == nil {
			return nil
		}
		if !isTopicNotReady(err) {
			break
		}
		// The topic is being auto-created; back off briefly and retry.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 200 * time.Millisecond):
		}
	}
	return fmt.Errorf("write kafka message: %w", err)
}

// isTopicNotReady reports whether err is the transient error seen while a topic
// is still being auto-created, and is therefore worth retrying.
func isTopicNotReady(err error) bool {
	return errors.Is(err, kafka.UnknownTopicOrPartition) ||
		errors.Is(err, kafka.LeaderNotAvailable)
}

// Close flushes and closes the underlying writer.
func (wr *Writer) Close() error { return wr.w.Close() }
