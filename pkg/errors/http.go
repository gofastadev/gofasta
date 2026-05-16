package apperrors

import "net/http"

// HTTPStatus maps an AppError type to the appropriate HTTP status code.
func HTTPStatus(e *AppError) int {
	switch e.Type {
	case NotFound:
		return http.StatusNotFound
	case Validation:
		return http.StatusUnprocessableEntity
	case Conflict:
		return http.StatusConflict
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case BadRequest:
		return http.StatusBadRequest
	case PreconditionFailed:
		return http.StatusPreconditionFailed
	case PreconditionRequired:
		return http.StatusPreconditionRequired
	default:
		return http.StatusInternalServerError
	}
}
