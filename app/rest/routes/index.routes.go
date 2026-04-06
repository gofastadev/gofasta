package routes

import (
	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/pkg/httputil"
)

// RouteConfig holds all controllers needed for route registration.
type RouteConfig struct {
	UserController   *controllers.UserController
	HealthController *controllers.HealthController
}

func InitApiRoutes(config *RouteConfig) *mux.Router {
	r := mux.NewRouter()

	// Health check at root level
	if config.HealthController != nil {
		r.HandleFunc("/health", httputil.Handle(config.HealthController.Check)).Methods("GET")
	}

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()
	UserRoutes(api, config.UserController)

	// Keep backward-compatible routes at root level
	UserRoutes(r, config.UserController)

	return r
}
