package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/LilianMrt/go-listings-service/internal/db"
	"github.com/LilianMrt/go-listings-service/internal/httpapi"
	"github.com/LilianMrt/go-listings-service/internal/listing"
	"github.com/LilianMrt/go-listings-service/internal/listing/store"
	"github.com/LilianMrt/go-listings-service/internal/observability"
	"github.com/LilianMrt/go-listings-service/internal/testsupport"
)

// TestAPIIntegration drives the real router (Huma -> service -> Postgres) over
// HTTP, covering the full listing lifecycle and key error cases.
func TestAPIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn := testsupport.StartPostgres(t)

	pool, err := db.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	health := observability.NewHealth()
	health.SetReady(true)
	router := httpapi.NewRouter(httpapi.Deps{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Health:   health,
		Listings: listing.NewService(store.NewPostgres(pool)),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	c := &apiClient{t: t, base: srv.URL}

	seller := uuid.NewString()
	var id string

	t.Run("openapi spec is served", func(t *testing.T) {
		status, body := c.raw(http.MethodGet, "/openapi.json", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if !bytes.Contains(body, []byte(`"openapi"`)) {
			t.Fatalf("body does not look like an OpenAPI document")
		}
	})

	t.Run("create returns 201 draft with defaults", func(t *testing.T) {
		status, body := c.do(http.MethodPost, "/v1/listings", map[string]any{
			"title": "Sunny 2-room", "price_cents": 250000, "city": "Toulouse",
			"postal_code": "31000", "surface_m2": 45, "rooms": 2, "seller_id": seller,
		})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body: %v)", status, body)
		}
		if body["status"] != "draft" {
			t.Errorf("status = %v, want draft", body["status"])
		}
		if body["currency"] != "EUR" {
			t.Errorf("currency = %v, want EUR", body["currency"])
		}
		id, _ = body["id"].(string)
		if id == "" {
			t.Fatal("no id returned")
		}
	})

	t.Run("create with invalid fields returns 422", func(t *testing.T) {
		status, _ := c.do(http.MethodPost, "/v1/listings", map[string]any{
			"title": "", "price_cents": 0, "city": "Lyon", "postal_code": "69000", "seller_id": seller,
		})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", status)
		}
	})

	t.Run("get returns the listing", func(t *testing.T) {
		status, body := c.do(http.MethodGet, "/v1/listings/"+id, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if body["city"] != "Toulouse" {
			t.Errorf("city = %v, want Toulouse", body["city"])
		}
	})

	t.Run("patch updates price", func(t *testing.T) {
		status, body := c.do(http.MethodPatch, "/v1/listings/"+id, map[string]any{"price_cents": 260000})
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if body["price_cents"].(float64) != 260000 {
			t.Errorf("price_cents = %v, want 260000", body["price_cents"])
		}
	})

	t.Run("selling a draft is rejected with 409", func(t *testing.T) {
		status, _ := c.do(http.MethodPost, "/v1/listings/"+id+"/sell", nil)
		if status != http.StatusConflict {
			t.Fatalf("status = %d, want 409", status)
		}
	})

	t.Run("publish then sell succeed", func(t *testing.T) {
		if status, _ := c.do(http.MethodPost, "/v1/listings/"+id+"/publish", nil); status != http.StatusOK {
			t.Fatalf("publish status = %d, want 200", status)
		}
		status, body := c.do(http.MethodPost, "/v1/listings/"+id+"/sell", nil)
		if status != http.StatusOK {
			t.Fatalf("sell status = %d, want 200", status)
		}
		if body["status"] != "sold" {
			t.Errorf("status = %v, want sold", body["status"])
		}
	})

	t.Run("list filters by city and status", func(t *testing.T) {
		status, body := c.do(http.MethodGet, "/v1/listings?city=Toulouse&status=sold&limit=10", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if body["count"].(float64) < 1 {
			t.Errorf("count = %v, want >= 1", body["count"])
		}
	})

	t.Run("delete then get returns 404", func(t *testing.T) {
		if status, _ := c.do(http.MethodDelete, "/v1/listings/"+id, nil); status != http.StatusNoContent {
			t.Fatalf("delete status = %d, want 204", status)
		}
		if status, _ := c.do(http.MethodGet, "/v1/listings/"+id, nil); status != http.StatusNotFound {
			t.Fatalf("get status = %d, want 404", status)
		}
	})
}

// apiClient is a tiny JSON HTTP client for the test server.
type apiClient struct {
	t    *testing.T
	base string
}

// do sends an optional JSON body and decodes a JSON object response.
func (c *apiClient) do(method, path string, payload any) (int, map[string]any) {
	status, raw := c.raw(method, path, payload)
	if len(raw) == 0 {
		return status, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		c.t.Fatalf("decode %s %s response: %v (raw: %s)", method, path, err, raw)
	}
	return status, out
}

// raw sends a request and returns the status code and raw body bytes.
func (c *apiClient) raw(method, path string, payload any) (int, []byte) {
	c.t.Helper()
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			c.t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, raw
}
