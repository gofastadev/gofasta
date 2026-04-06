package apperrors

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// safeErrorPrefixes are error message prefixes that are safe to expose to clients.
var safeErrorPrefixes = []string{
	"not found",
	"validation failed",
	"invalid",
	"already exists",
	"unauthorized",
	"forbidden",
	"authentication required",
	"permission denied",
}

// GraphQLErrorPresenter sanitizes errors before sending them to the client.
// Known/safe errors pass through; unknown errors are logged and replaced with a generic message.
func GraphQLErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	defaultErr := graphql.DefaultErrorPresenter(ctx, err)
	msg := defaultErr.Message

	// If it's an AppError, use its message (it's designed to be user-facing)
	var appErr *AppError
	if errors.As(err, &appErr) {
		defaultErr.Message = appErr.Message
		return defaultErr
	}

	// Check if the error message starts with a known safe prefix
	lowerMsg := strings.ToLower(msg)
	for _, prefix := range safeErrorPrefixes {
		if strings.HasPrefix(lowerMsg, prefix) {
			return defaultErr
		}
	}

	// Log the internal error and return a generic message
	slog.Error("graphql internal error", "error", msg, "path", defaultErr.Path)
	defaultErr.Message = "an internal error occurred"
	return defaultErr
}
