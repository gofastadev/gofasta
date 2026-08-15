package health

import (
	"net/http"
	"time"

	"github.com/gofastadev/gofasta/pkg/cache"
	"github.com/gofastadev/gofasta/pkg/httputil"
	"gorm.io/gorm"
)

var startTime = time.Now()

// Controller handles health check endpoints.
type Controller struct {
	DB    *gorm.DB
	Cache cache.CacheService
	// ExpectedDriver, when non-empty, is the configured database driver
	// name (config.Database.Driver). Ready compares it against the live
	// connection's dialector so a degraded-mode fallback (e.g. an
	// in-memory sqlite stand-in for an unreachable postgres) reports as
	// down instead of answering "up" against the wrong database.
	ExpectedDriver string
}

// NewController creates a new health Controller. Cache can be nil.
// The degraded-driver check is disabled; use NewControllerForDriver to
// enable it.
func NewController(db *gorm.DB, cacheService cache.CacheService) *Controller {
	return NewControllerForDriver(db, cacheService, "")
}

// NewControllerForDriver creates a health Controller that also verifies
// the live connection uses the expected driver (see ExpectedDriver).
// Pass config.Database.Driver; an empty string disables the check.
func NewControllerForDriver(db *gorm.DB, cacheService cache.CacheService, expectedDriver string) *Controller {
	return &Controller{DB: db, Cache: cacheService, ExpectedDriver: expectedDriver}
}

// databaseStatus classifies the database dependency for Ready. It never
// panics: a nil DB or a connection-pool-less *gorm.DB (the last-resort
// fallback some scaffolds construct when the real database is
// unreachable) reports down instead of nil-dereferencing, and a live
// connection on the wrong driver — degraded-fallback mode — also
// reports down so readiness can never pass against a stand-in store.
func (h *Controller) databaseStatus(r *http.Request) (status, errMsg string) {
	if h.DB == nil {
		return "down", "no database connection"
	}
	sqlDB, err := h.DB.DB()
	if err != nil || sqlDB == nil {
		return "down", "no usable connection pool"
	}
	if err := sqlDB.PingContext(r.Context()); err != nil {
		return "down", err.Error()
	}
	if h.ExpectedDriver != "" && h.DB.Name() != h.ExpectedDriver {
		return "down", "degraded: connected driver " + h.DB.Name() +
			" does not match configured driver " + h.ExpectedDriver
	}
	return "up", ""
}

// Check handles GET /health — basic liveness.
func (h *Controller) Check(w http.ResponseWriter, r *http.Request) error {
	return httputil.OK(w, map[string]string{"status": "up"})
}

// Live handles GET /health/live — process is alive.
func (h *Controller) Live(w http.ResponseWriter, r *http.Request) error {
	return httputil.OK(w, map[string]string{"status": "up"})
}

// Ready handles GET /health/ready — can serve traffic (checks dependencies).
func (h *Controller) Ready(w http.ResponseWriter, r *http.Request) error {
	checks := []map[string]interface{}{}

	dbCheck := map[string]interface{}{"name": "database"}
	start := time.Now()
	if status, errMsg := h.databaseStatus(r); errMsg != "" {
		dbCheck["status"] = status
		dbCheck["error"] = errMsg
	} else {
		dbCheck["status"] = status
	}
	dbCheck["duration"] = time.Since(start).String()
	checks = append(checks, dbCheck)

	if h.Cache != nil {
		cacheCheck := map[string]interface{}{"name": "cache"}
		start = time.Now()
		if err := h.Cache.Ping(r.Context()); err != nil {
			cacheCheck["status"] = "down"
			cacheCheck["error"] = err.Error()
		} else {
			cacheCheck["status"] = "up"
		}
		cacheCheck["duration"] = time.Since(start).String()
		checks = append(checks, cacheCheck)
	}

	overallStatus := "up"
	for _, c := range checks {
		if c["status"] == "down" {
			overallStatus = "down"
			break
		}
	}

	statusCode := http.StatusOK
	if overallStatus == "down" {
		statusCode = http.StatusServiceUnavailable
	}

	return httputil.JSON(w, statusCode, map[string]interface{}{
		"status": overallStatus,
		"uptime": time.Since(startTime).String(),
		"checks": checks,
	})
}
