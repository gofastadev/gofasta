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
// If the handler returns an *AppError, it writes a structured JSON error
// response; a 5xx AppError is also logged with its wrapped cause, which the
// body never exposes. Unknown errors are logged and a generic 500 is returned.
func Handle(h AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			var appErr *apperrors.AppError
			if errors.As(err, &appErr) {
				status := apperrors.HTTPStatus(appErr)
				// A 5xx is the server admitting it broke, and the reason is
				// almost always in AppError.Internal — which the response body
				// deliberately does not carry. Logging only the non-AppError
				// branch meant every `NewInternal(msg, cause)` produced a 500
				// with NO log line at all and the cause discarded: the one
				// failure that most needs explaining was the one that left no
				// trace. 4xx stay unlogged; they are the caller's to fix and
				// logging them is noise.
				if status >= http.StatusInternalServerError {
					slog.Error("request failed",
						"error", appErr.Error(), "method", r.Method, "path", r.URL.Path)
				}
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
