package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"

	"github.com/LilianMrt/go-listings-service/internal/db"
	"github.com/LilianMrt/go-listings-service/internal/events"
	"github.com/LilianMrt/go-listings-service/internal/httpapi"
	"github.com/LilianMrt/go-listings-service/internal/listing"
	"github.com/LilianMrt/go-listings-service/internal/listing/store"
	"github.com/LilianMrt/go-listings-service/internal/observability"
	"github.com/LilianMrt/go-listings-service/internal/testsupport"
)

// TestListingEventsPublished drives a real create through the router and asserts
// that a listing.created event is produced to Kafka, keyed by the listing id.
func TestListingEventsPublished(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const topic = "listings.events"

	dsn := testsupport.StartPostgres(t)
	brokers := testsupport.StartKafka(t)

	pool, err := db.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	writer := events.NewWriter(brokers, topic)
	t.Cleanup(func() { _ = writer.Close() })

	health := observability.NewHealth()
	health.SetReady(true)
	router := httpapi.NewRouter(httpapi.Deps{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Health:   health,
		Listings: listing.NewService(store.NewPostgres(pool), listing.WithPublisher(writer)),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	c := &apiClient{t: t, base: srv.URL}

	seller := uuid.NewString()
	status, body := c.do(http.MethodPost, "/v1/listings", map[string]any{
		"title": "Event flat", "price_cents": 300000, "city": "Nantes",
		"postal_code": "44000", "surface_m2": 60, "rooms": 3, "seller_id": seller,
	})
	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (body: %v)", status, body)
	}
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("no id returned")
	}

	// Consume the event back from the topic and assert its shape.
	reader := segmentio.NewReader(segmentio.ReaderConfig{
		Brokers:   brokers,
		Topic:     topic,
		Partition: 0,
		MaxWait:   500 * time.Millisecond,
	})
	t.Cleanup(func() { _ = reader.Close() })
	if err := reader.SetOffset(segmentio.FirstOffset); err != nil {
		t.Fatalf("set offset: %v", err)
	}

	readCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("read event: %v", err)
	}

	if string(msg.Key) != id {
		t.Errorf("message key = %q, want listing id %q", msg.Key, id)
	}

	var ev struct {
		Type       string    `json:"type"`
		ID         string    `json:"id"`
		OccurredAt time.Time `json:"occurred_at"`
		Data       struct {
			City   string `json:"city"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		t.Fatalf("decode event: %v (raw: %s)", err, msg.Value)
	}
	if ev.Type != string(listing.EventCreated) {
		t.Errorf("type = %q, want %q", ev.Type, listing.EventCreated)
	}
	if ev.ID != id {
		t.Errorf("event id = %q, want %q", ev.ID, id)
	}
	if ev.OccurredAt.IsZero() {
		t.Error("occurred_at is zero")
	}
	if ev.Data.City != "Nantes" {
		t.Errorf("data.city = %q, want Nantes", ev.Data.City)
	}
	if ev.Data.Status != string(listing.StatusDraft) {
		t.Errorf("data.status = %q, want draft", ev.Data.Status)
	}
}
