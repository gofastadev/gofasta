package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockCache implements cache.CacheService for testing.
type mockCache struct {
	pingErr error
}

func (m *mockCache) Get(_ context.Context, _ string) (string, error)                     { return "", nil }
func (m *mockCache) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error { return nil }
func (m *mockCache) Delete(_ context.Context, _ string) error                              { return nil }
func (m *mockCache) Flush(_ context.Context) error                                         { return nil }
func (m *mockCache) Ping(_ context.Context) error                                          { return m.pingErr }

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	return db
}

func TestNewController(t *testing.T) {
	db := newTestDB(t)
	c := NewController(db, &mockCache{})
	if c == nil {
		t.Fatal("expected non-nil controller")
	}
	if c.DB != db {
		t.Error("expected DB to be set")
	}
}

func TestCheck(t *testing.T) {
	c := NewController(newTestDB(t), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	err := c.Check(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "up" {
		t.Errorf("expected status 'up', got %q", body["status"])
	}
}

func TestLive(t *testing.T) {
	c := NewController(newTestDB(t), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	err := c.Live(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "up" {
		t.Errorf("expected status 'up', got %q", body["status"])
	}
}

func TestReady_AllUp(t *testing.T) {
	db := newTestDB(t)
	cache := &mockCache{pingErr: nil}
	c := NewController(db, cache)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	err := c.Ready(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "up" {
		t.Errorf("expected overall status 'up', got %v", body["status"])
	}

	checks, ok := body["checks"].([]interface{})
	if !ok {
		t.Fatal("expected checks array")
	}
	if len(checks) != 2 {
		t.Errorf("expected 2 checks (db + cache), got %d", len(checks))
	}
}

func TestReady_CacheDown(t *testing.T) {
	db := newTestDB(t)
	cache := &mockCache{pingErr: context.DeadlineExceeded}
	c := NewController(db, cache)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	err := c.Ready(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "down" {
		t.Errorf("expected overall status 'down', got %v", body["status"])
	}
}

func TestReady_DBDown(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	// Close the underlying connection to simulate DB down
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.Close()

	c := NewController(db, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	err = c.Ready(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "down" {
		t.Errorf("expected overall status 'down', got %v", body["status"])
	}
}

func TestReady_NilCache(t *testing.T) {
	db := newTestDB(t)
	c := NewController(db, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	err := c.Ready(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	checks, ok := body["checks"].([]interface{})
	if !ok {
		t.Fatal("expected checks array")
	}
	if len(checks) != 1 {
		t.Errorf("expected 1 check (db only), got %d", len(checks))
	}
}
