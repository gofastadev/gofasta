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

// GraphQLErrorPresenter sanitizes errors before sending them to the client and
// classifies them for it.
//
// Two things happen here. The message is sanitized: an AppError's message is
// written by the application for a human to read, so it passes through; a bare
// error's message might be a connection string or a SQL fragment, so it passes
// through only when it starts with a known-safe prefix and is otherwise logged
// and replaced. Then the error is classified — every error the client receives
// carries a `code` extension naming its kind, and an AppError carrying Details
// carries them through as a `details` extension so field-level validation
// failures survive the trip.
//
// Register it with gqlgen via srv.SetErrorPresenter.
func GraphQLErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	defaultErr := graphql.DefaultErrorPresenter(ctx, err)
	if defaultErr == nil {
		return nil
	}
	msg := defaultErr.Message

	// If it's an AppError, use its message (it's designed to be user-facing)
	// and classify from its type.
	var appErr *AppError
	if errors.As(err, &appErr) {
		defaultErr.Message = appErr.Message
		setExtension(defaultErr, "code", appErr.Type.Code())
		if appErr.Details != nil {
			setExtension(defaultErr, "details", appErr.Details)
		}
		return defaultErr
	}

	// Check if the error message starts with a known safe prefix
	lowerMsg := strings.ToLower(msg)
	for _, prefix := range safeErrorPrefixes {
		if strings.HasPrefix(lowerMsg, prefix) {
			// Safe to show, but nothing classified it: the taxonomy lives on
			// AppError, and a bare error never reached it. INTERNAL is the
			// honest answer — anything else would invent a classification the
			// application never made.
			setExtension(defaultErr, "code", CodeInternal)
			return defaultErr
		}
	}

	// Log the internal error and return a generic message
	slog.Error("graphql internal error", "error", msg, "path", defaultErr.Path)
	defaultErr.Message = "an internal error occurred"
	setExtension(defaultErr, "code", CodeInternal)
	return defaultErr
}

// setExtension writes key without clobbering extensions the resolver already
// attached, allocating the map on first use.
func setExtension(err *gqlerror.Error, key string, value any) {
	if err.Extensions == nil {
		err.Extensions = map[string]any{}
	}
	err.Extensions[key] = value
}
