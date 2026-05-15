package httputil

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// errReader is an io.Reader that always errors on Read. Used to drive
// BindForm's ParseForm-error branch — once ParseForm sees a body with
// non-zero ContentLength but unreadable contents, it surfaces the read
// error and BindForm wraps it as a BadRequest.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read error") }

func TestBindForm_ParseFormError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(errReader{}))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = 10

	type FormData struct {
		Name string `schema:"name"`
	}
	_, err := BindForm[FormData](req)
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.BadRequest, appErr.Type)
}

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
