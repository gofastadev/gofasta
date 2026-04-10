package httputil

import (
	"errors"
	"log/slog"
	"net/http"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
)

// AppHandler is an http.HandlerFunc that returns an error.
// Errors are handled centrally by the Handle adapter.
type AppHandler func(w http.ResponseWriter, r *http.Request) error

// Handle converts an AppHandler into a standard http.HandlerFunc.
// If the handler returns an *AppError, it writes a structured JSON error response.
// Unknown errors are logged and a generic 500 response is returned.
func Handle(h AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			var appErr *apperrors.AppError
			if errors.As(err, &appErr) {
				status := apperrors.HTTPStatus(appErr)
				response := map[string]interface{}{
					"error": appErr.Message,
				}
				if appErr.Details != nil {
					response["details"] = appErr.Details
				}
				_ = JSON(w, status, response)
			} else {
				slog.Error("unhandled error", "error", err, "method", r.Method, "path", r.URL.Path)
				_ = JSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
			}
		}
	}
}
