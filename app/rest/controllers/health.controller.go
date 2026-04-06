package controllers

import (
	"net/http"

	"github.com/healtronlabs/gofasta/pkg/httputil"
	"gorm.io/gorm"
)

// HealthController handles health check endpoints.
type HealthController struct {
	DB *gorm.DB
}

// NewHealthController creates a new HealthController.
func NewHealthController(db *gorm.DB) *HealthController {
	return &HealthController{DB: db}
}

// Check handles GET /health requests.
func (h *HealthController) Check(w http.ResponseWriter, r *http.Request) error {
	sqlDB, err := h.DB.DB()
	if err != nil {
		return httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "failed to get database instance",
		})
	}

	if err := sqlDB.PingContext(r.Context()); err != nil {
		return httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "database ping failed",
		})
	}

	return httputil.OK(w, map[string]string{"status": "healthy"})
}
