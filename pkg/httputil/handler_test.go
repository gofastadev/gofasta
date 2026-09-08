package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestHandle_SuccessfulHandler(t *testing.T) {
	h := Handle(func(w http.ResponseWriter, r *http.Request) error {
		return OK(w, map[string]string{"result": "success"})
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "success", body["result"])
}

func TestHandle_AppError(t *testing.T) {
	tests := []struct {
		name           string
		err            *apperrors.AppError
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "not found",
			err:            apperrors.NewNotFound("user not found", nil),
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "user not found",
		},
		{
			name:           "bad request",
			err:            apperrors.NewBadRequest("invalid input", nil),
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid input",
		},
		{
			name:           "unauthorized",
			err:            apperrors.NewUnauthorized("not authenticated", nil),
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "not authenticated",
		},
		{
			name:           "forbidden",
			err:            apperrors.NewForbidden("access denied", nil),
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "access denied",
		},
		{
			name:           "conflict",
			err:            apperrors.NewConflict("already exists", nil),
			expectedStatus: http.StatusConflict,
			expectedMsg:    "already exists",
		},
		{
			name:           "validation",
			err:            apperrors.NewValidation("validation failed", []string{"field required"}),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedMsg:    "validation failed",
		},
		{
			name:           "internal",
			err:            apperrors.NewInternal("something broke", nil),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "something broke",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handle(func(w http.ResponseWriter, r *http.Request) error {
				return tt.err
			})

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var body map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &body)
			assert.Equal(t, tt.expectedMsg, body["error"])
		})
	}
}

func TestHandle_AppErrorWithDetails(t *testing.T) {
	details := []map[string]string{{"field": "email", "message": "required"}}
	appErr := apperrors.NewValidation("validation failed", details)

	h := Handle(func(w http.ResponseWriter, r *http.Request) error {
		return appErr
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "validation failed", body["error"])
	assert.NotNil(t, body["details"])
}

func TestHandle_UnknownError(t *testing.T) {
	h := Handle(func(w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("unexpected database error")
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "Internal Server Error", body["error"])
}

func TestHandle_NilError(t *testing.T) {
	h := Handle(func(w http.ResponseWriter, r *http.Request) error {
		return OK(w, map[string]string{"ok": "true"})
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

// A 500 that logs nothing is a 500 nobody can debug: the body deliberately
// carries only AppError.Message, so the wrapped cause exists in exactly one
// place — the log. Before 2026-09-08 only the non-AppError branch logged, so
// every `NewInternal(msg, cause)` answered 500 and discarded the cause.
func TestHandle_LogsServerErrorsWithTheirCause(t *testing.T) {
	var buf bytes.Buffer
	restore := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(restore)

	cause := errors.New("decrypt: cipher: message authentication failed")
	h := Handle(func(w http.ResponseWriter, r *http.Request) error {
		return apperrors.NewInternal("slack app operation failed", cause)
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/slack_apps/x/channels", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	logged := buf.String()
	assert.Contains(t, logged, "request failed")
	assert.Contains(t, logged, "cipher: message authentication failed")
	assert.Contains(t, logged, "/slack_apps/x/channels")
	// The cause stays out of the response — it is for operators, not callers.
	assert.NotContains(t, rec.Body.String(), "cipher")
}

// 4xx are the caller's to fix; logging each one turns a validation typo into
// an operator alert.
func TestHandle_DoesNotLogClientErrors(t *testing.T) {
	var buf bytes.Buffer
	restore := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(restore)

	h := Handle(func(w http.ResponseWriter, r *http.Request) error {
		return apperrors.NewNotFound("user not found", errors.New("no rows"))
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users/x", nil))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, buf.String())
}
