package apperrors

import (
	stderrors "errors"
	"strings"
	"sync"
)

// Classifying string errors
//
// Most codebases that adopt typed errors adopt them gradually: the transport
// layer returns *AppError, while the service layer underneath still returns
// errors.New("course not found") from a thousand places. Something has to turn
// the second into the first, and doing it at each call site means the same
// wording is classified two different ways in two different files — which
// hands the same client contradictory codes for the same failure.
//
// Classify does it in one place. The default rules below match on wording that
// is conventional across Go codebases rather than specific to any project;
// RegisterClassifier adds project wording without forking the mechanism.

// classifier maps a lowercase substring to the error it implies.
type classifier struct {
	match string
	build func(msg string) *AppError
}

// defaultClassifiers are ordered most-specific first: "already enrolled" must
// be seen before the bare "already", and the authentication phrasings before
// the authorization ones, because "unauthorized" appears inside sentences that
// mean either.
var defaultClassifiers = []classifier{
	{"authentication required", func(m string) *AppError { return NewUnauthorized(m, nil) }},
	{"unauthenticated", func(m string) *AppError { return NewUnauthorized(m, nil) }},
	{"not authenticated", func(m string) *AppError { return NewUnauthorized(m, nil) }},
	{"invalid credentials", func(m string) *AppError { return NewUnauthorized(m, nil) }},
	{"token expired", func(m string) *AppError { return NewUnauthorized(m, nil) }},
	{"unauthorized", func(m string) *AppError { return NewUnauthorized(m, nil) }},

	{"permission denied", func(m string) *AppError { return NewForbidden(m, nil) }},
	{"do not have permission", func(m string) *AppError { return NewForbidden(m, nil) }},
	{"access denied", func(m string) *AppError { return NewForbidden(m, nil) }},
	{"not allowed", func(m string) *AppError { return NewForbidden(m, nil) }},
	{"forbidden", func(m string) *AppError { return NewForbidden(m, nil) }},

	{"not found", func(m string) *AppError { return NewNotFound(m, nil) }},
	{"does not exist", func(m string) *AppError { return NewNotFound(m, nil) }},
	{"no such", func(m string) *AppError { return NewNotFound(m, nil) }},

	{"already exists", func(m string) *AppError { return NewConflict(m, nil) }},
	{"duplicate", func(m string) *AppError { return NewConflict(m, nil) }},
	{"conflict", func(m string) *AppError { return NewConflict(m, nil) }},
	{"already", func(m string) *AppError { return NewConflict(m, nil) }},

	{"validation failed", func(m string) *AppError { return NewValidation(m, nil) }},
	{"is required", func(m string) *AppError { return NewValidation(m, nil) }},
	{"cannot be empty", func(m string) *AppError { return NewValidation(m, nil) }},
	{"must be", func(m string) *AppError { return NewValidation(m, nil) }},
	{"invalid", func(m string) *AppError { return NewValidation(m, nil) }},
}

var (
	classifierMu    sync.RWMutex
	extraClassifier []classifier
)

// RegisterClassifier teaches Classify a project-specific phrase.
//
// Registered rules are consulted before the defaults, so a project can both
// add wording and override a default it disagrees with. Match is compared
// case-insensitively as a substring.
//
// Call it at startup, before serving. It is safe to call concurrently, but a
// rule registered while requests are in flight applies only to what follows.
func RegisterClassifier(match string, build func(msg string) *AppError) {
	if match == "" || build == nil {
		return
	}
	classifierMu.Lock()
	defer classifierMu.Unlock()
	extraClassifier = append(extraClassifier, classifier{
		match: strings.ToLower(match),
		build: build,
	})
}

// ResetClassifiers drops every rule added by RegisterClassifier. Intended for
// tests, which would otherwise leak rules into one another.
func ResetClassifiers() {
	classifierMu.Lock()
	defer classifierMu.Unlock()
	extraClassifier = nil
}

// Classify turns a bare error message into a typed *AppError.
//
// Anything unrecognized becomes Internal, deliberately. The GraphQL and HTTP
// presenters replace an Internal message with a generic one, and text the API
// never deliberately shaped for a client is exactly the text that should not
// reach one — a driver error naming a column, a path, a query. A message that
// deserves better deserves a rule.
func Classify(msg string) *AppError {
	lower := strings.ToLower(msg)

	classifierMu.RLock()
	extras := extraClassifier
	classifierMu.RUnlock()

	for _, c := range extras {
		if strings.Contains(lower, c.match) {
			return c.build(msg)
		}
	}
	for _, c := range defaultClassifiers {
		if strings.Contains(lower, c.match) {
			return c.build(msg)
		}
	}
	return NewInternal(msg, nil)
}

// ClassifyError is Classify over an error value.
//
// An error that is already an *AppError is returned unchanged — classifying it
// again would discard the type the producer chose, along with any Details it
// attached, in favor of a guess made from its message.
func ClassifyError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr
	}
	return Classify(err.Error())
}
