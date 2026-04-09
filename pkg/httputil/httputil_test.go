package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- Response helpers ----------

func TestJSON_WritesStatusAndBody(t *testing.T) {
	tests := []struct {
		name   string
		status int
		data   interface{}
	}{
		{"ok with map", http.StatusOK, map[string]string{"msg": "hello"}},
		{"created with struct", http.StatusCreated, struct {
			ID int `json:"id"`
		}{ID: 42}},
		{"bad request", http.StatusBadRequest, map[string]string{"error": "bad"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			err := JSON(rec, tt.status, tt.data)
			require.NoError(t, err)

			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var got map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &got)
			require.NoError(t, err)
		})
	}
}

func TestOK_Returns200(t *testing.T) {
	rec := httptest.NewRecorder()
	err := OK(rec, map[string]string{"status": "ok"})
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "ok", body["status"])
}

func TestCreated_Returns201(t *testing.T) {
	rec := httptest.NewRecorder()
	err := Created(rec, map[string]int{"id": 1})
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]int
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, 1, body["id"])
}

func TestNoContent_Returns204(t *testing.T) {
	rec := httptest.NewRecorder()
	err := NoContent(rec)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

// ---------- Handle adapter ----------

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

// ---------- BindJSON ----------

type testPayload struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0"`
}

func TestBindJSON_ValidPayload(t *testing.T) {
	body := `{"name":"Alice","email":"alice@example.com","age":30}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	result, err := BindJSON[testPayload](req)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, "alice@example.com", result.Email)
	assert.Equal(t, 30, result.Age)
}

func TestBindJSON_InvalidJSON(t *testing.T) {
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	_, err := BindJSON[testPayload](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.BadRequest, appErr.Type)
	assert.Equal(t, "invalid request payload", appErr.Message)
}

func TestBindJSON_ValidationFailure(t *testing.T) {
	// Missing required "name" field
	body := `{"email":"alice@example.com","age":30}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	_, err := BindJSON[testPayload](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.Validation, appErr.Type)
	assert.Equal(t, "validation failed", appErr.Message)
	assert.NotNil(t, appErr.Details)
}

func TestBindJSON_InvalidEmail(t *testing.T) {
	body := `{"name":"Alice","email":"not-an-email","age":30}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	_, err := BindJSON[testPayload](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.Validation, appErr.Type)
}

func TestBindJSON_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	_, err := BindJSON[testPayload](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.BadRequest, appErr.Type)
}

// ---------- BindQuery ----------

type testQueryParams struct {
	Page  int    `schema:"page" validate:"gte=1"`
	Limit int    `schema:"limit" validate:"gte=1,lte=100"`
	Sort  string `schema:"sort"`
}

func TestBindQuery_ValidQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=2&limit=25&sort=name", nil)

	result, err := BindQuery[testQueryParams](req)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 25, result.Limit)
	assert.Equal(t, "name", result.Sort)
}

func TestBindQuery_ValidationFailure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=0&limit=25", nil)

	_, err := BindQuery[testQueryParams](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.Validation, appErr.Type)
}

func TestBindQuery_EmptyQuery(t *testing.T) {
	// page and limit will be 0, which violates gte=1
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := BindQuery[testQueryParams](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.Validation, appErr.Type)
}

type testOptionalQuery struct {
	Search string `schema:"search"`
	Active bool   `schema:"active"`
}

func TestBindQuery_OptionalFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?search=hello&active=true", nil)

	result, err := BindQuery[testOptionalQuery](req)
	require.NoError(t, err)
	assert.Equal(t, "hello", result.Search)
	assert.True(t, result.Active)
}

func TestBindQuery_MissingOptionalFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	result, err := BindQuery[testOptionalQuery](req)
	require.NoError(t, err)
	assert.Empty(t, result.Search)
	assert.False(t, result.Active)
}

// ---------- BindForm ----------

type testFormData struct {
	Username string `schema:"username" validate:"required"`
	Password string `schema:"password" validate:"required,min=8"`
}

func TestBindForm_ValidForm(t *testing.T) {
	body := "username=alice&password=secret1234"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := BindForm[testFormData](req)
	require.NoError(t, err)
	assert.Equal(t, "alice", result.Username)
	assert.Equal(t, "secret1234", result.Password)
}

func TestBindForm_ValidationFailure(t *testing.T) {
	body := "username=alice&password=short"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := BindForm[testFormData](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.Validation, appErr.Type)
}

func TestBindForm_MissingRequiredField(t *testing.T) {
	body := "password=secret1234"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := BindForm[testFormData](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.Validation, appErr.Type)
}

func TestBindForm_EmptyForm(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := BindForm[testFormData](req)
	require.Error(t, err)
}

// ---------- JSON edge cases ----------

func TestJSON_NilData(t *testing.T) {
	rec := httptest.NewRecorder()
	err := JSON(rec, http.StatusOK, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "null")
}

func TestJSON_SliceData(t *testing.T) {
	rec := httptest.NewRecorder()
	data := []string{"a", "b", "c"}
	err := JSON(rec, http.StatusOK, data)
	require.NoError(t, err)

	var result []string
	json.Unmarshal(rec.Body.Bytes(), &result)
	assert.Equal(t, data, result)
}

// ---------- BindQuery decode error ----------

func TestBindQuery_DecodeError(t *testing.T) {
	// Use a struct with a type that can't be decoded from query string
	type badQuery struct {
		Count int `schema:"count" validate:"gte=0"`
	}
	req := httptest.NewRequest(http.MethodGet, "/?count=not_a_number", nil)

	_, err := BindQuery[badQuery](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.BadRequest, appErr.Type)
}

// ---------- BindForm decode error ----------

func TestBindForm_DecodeError(t *testing.T) {
	type badForm struct {
		Count int `schema:"count" validate:"gte=0"`
	}
	body := "count=not_a_number"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := BindForm[badForm](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.BadRequest, appErr.Type)
}
