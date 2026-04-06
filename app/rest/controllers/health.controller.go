package controllers

import (
	"net/http"
	"time"

	"github.com/healtronlabs/gofasta/pkg/cache"
	"github.com/healtronlabs/gofasta/pkg/httputil"
	"gorm.io/gorm"
)

var startTime = time.Now()

// HealthController handles health check endpoints.
type HealthController struct {
	DB    *gorm.DB
	Cache cache.CacheService
}

// NewHealthController creates a new HealthController. Cache can be nil.
func NewHealthController(db *gorm.DB, cacheService cache.CacheService) *HealthController {
	return &HealthController{DB: db, Cache: cacheService}
}

// Check handles GET /health — basic liveness.
func (h *HealthController) Check(w http.ResponseWriter, r *http.Request) error {
	return httputil.OK(w, map[string]string{"status": "up"})
}

// Live handles GET /health/live — process is alive.
func (h *HealthController) Live(w http.ResponseWriter, r *http.Request) error {
	return httputil.OK(w, map[string]string{"status": "up"})
}

// Ready handles GET /health/ready — can serve traffic (checks dependencies).
func (h *HealthController) Ready(w http.ResponseWriter, r *http.Request) error {
	checks := []map[string]interface{}{}

	// Database check
	dbCheck := map[string]interface{}{"name": "database"}
	start := time.Now()
	if sqlDB, err := h.DB.DB(); err == nil {
		if err := sqlDB.PingContext(r.Context()); err != nil {
			dbCheck["status"] = "down"
			dbCheck["error"] = err.Error()
		} else {
			dbCheck["status"] = "up"
		}
	} else {
		dbCheck["status"] = "down"
		dbCheck["error"] = err.Error()
	}
	dbCheck["duration"] = time.Since(start).String()
	checks = append(checks, dbCheck)

	// Cache check (if configured)
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

	// Overall status
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
