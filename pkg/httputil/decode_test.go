package httputil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type decodeTestPayload struct {
	Name string `json:"name" schema:"name"`
	Age  int    `json:"age"  schema:"age"`
}

// TestDecodeJSON_Success — happy path: a well-formed JSON body decodes
// into T without invoking any validator. Custom validation tags on T
// (uuid4_valid etc.) would not be touched, even if registered globally
// elsewhere — this is the whole point of DecodeJSON vs BindJSON.
func TestDecodeJSON_Success(t *testing.T) {
	body := bytes.NewBufferString(`{"name":"Alice","age":30}`)
	req := httptest.NewRequest(http.MethodPost, "/x", body)

	got, err := DecodeJSON[decodeTestPayload](req)
	require.NoError(t, err)
	assert.Equal(t, "Alice", got.Name)
	assert.Equal(t, 30, got.Age)
}

// TestDecodeJSON_MalformedJSON — non-JSON returns an apperrors.BadRequest
// the error middleware can translate to 400.
func TestDecodeJSON_MalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString("not-json"))

	_, err := DecodeJSON[decodeTestPayload](req)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusBadRequest, apperrors.HTTPStatus(appErr))
}

// TestDecodeJSON_DoesNotPanicOnUnknownValidationTag — the regression
// driver for adding DecodeJSON. A struct with a custom-validator tag
// the caller's validator hasn't registered must NOT panic; DecodeJSON
// doesn't validate, so the tag is irrelevant during decode.
func TestDecodeJSON_DoesNotPanicOnUnknownValidationTag(t *testing.T) {
	type unknownTagPayload struct {
		ID string `json:"id" validate:"some_tag_not_registered_anywhere"`
	}
	body := bytes.NewBufferString(`{"id":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/x", body)

	got, err := DecodeJSON[unknownTagPayload](req)
	require.NoError(t, err)
	assert.Equal(t, "abc", got.ID)
}

// TestDecodeQuery_Success — query string decodes into typed fields via
// gorilla/schema; field names follow `schema:` tags.
func TestDecodeQuery_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?name=Bob&age=42", nil)

	got, err := DecodeQuery[decodeTestPayload](req)
	require.NoError(t, err)
	assert.Equal(t, "Bob", got.Name)
	assert.Equal(t, 42, got.Age)
}

// TestDecodeQuery_BadType — gorilla/schema returns an error when a
// numeric field gets a non-numeric value; DecodeQuery wraps it as
// BadRequest.
func TestDecodeQuery_BadType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?age=notanumber", nil)

	_, err := DecodeQuery[decodeTestPayload](req)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusBadRequest, apperrors.HTTPStatus(appErr))
}

// TestDecodeForm_Success — form data decodes into typed fields.
func TestDecodeForm_Success(t *testing.T) {
	form := strings.NewReader("name=Carol&age=25")
	req := httptest.NewRequest(http.MethodPost, "/x", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got, err := DecodeForm[decodeTestPayload](req)
	require.NoError(t, err)
	assert.Equal(t, "Carol", got.Name)
	assert.Equal(t, 25, got.Age)
}

// TestDecodeForm_ParseFormError — when the request body errors on read,
// r.ParseForm() surfaces the read error and DecodeForm wraps it as
// BadRequest. Drives the ParseForm-error branch that the happy-path
// and bad-type tests don't exercise. Reuses errReader from bind_test.go,
// which lives in the same package.
func TestDecodeForm_ParseFormError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", io.NopCloser(errReader{}))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = 10

	_, err := DecodeForm[decodeTestPayload](req)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusBadRequest, apperrors.HTTPStatus(appErr))
}

// TestDecodeForm_BadType — non-numeric value for an int field returns
// a BadRequest.
func TestDecodeForm_BadType(t *testing.T) {
	form := strings.NewReader("age=notanumber")
	req := httptest.NewRequest(http.MethodPost, "/x", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := DecodeForm[decodeTestPayload](req)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusBadRequest, apperrors.HTTPStatus(appErr))
}
