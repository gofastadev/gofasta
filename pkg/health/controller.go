// Package health provides health check endpoints for gofasta applications.
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
}

// NewController creates a new health Controller. Cache can be nil.
func NewController(db *gorm.DB, cacheService cache.CacheService) *Controller {
	return &Controller{DB: db, Cache: cacheService}
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
