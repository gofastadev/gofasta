package httputil

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
)

// DecodeJSON decodes a JSON request body into T and returns it.
//
// Unlike BindJSON, DecodeJSON does not validate the result — validation
// belongs at the controller boundary using the project's configured
// validator (typically validators.AppValidator), which has access to
// the project's custom validation functions (uuid4_valid,
// is_record_exist_by_*, etc.). BindJSON's hardcoded internal validator
// panics on those tags, so callers that depend on custom validation
// should use DecodeJSON + their own ValidateStruct call:
//
//	in, err := httputil.DecodeJSON[dtos.TUpdateUserDto](r)
//	if err != nil { return err }
//	if errs := c.validator.ValidateStruct(in); len(errs) > 0 {
//	    return apperrors.NewValidation("invalid input", errs)
//	}
//
// Returns an apperrors.BadRequest on malformed JSON so the framework
// error middleware can map it to a 400 response.
func DecodeJSON[T any](r *http.Request) (T, error) {
	var val T
	if err := json.NewDecoder(r.Body).Decode(&val); err != nil {
		return val, apperrors.NewBadRequest("invalid request payload", nil)
	}
	return val, nil
}

// DecodeQuery decodes URL query parameters into T and returns it.
// Like DecodeJSON, validation is the caller's responsibility — see the
// DecodeJSON doc for the recommended caller pattern.
func DecodeQuery[T any](r *http.Request) (T, error) {
	var val T
	if err := getDecoder().Decode(&val, r.URL.Query()); err != nil {
		return val, apperrors.NewBadRequest("invalid query parameters", nil)
	}
	return val, nil
}

// DecodeForm decodes form data into T and returns it. Validation is the
// caller's responsibility — see the DecodeJSON doc for the recommended
// caller pattern.
func DecodeForm[T any](r *http.Request) (T, error) {
	var val T
	if err := r.ParseForm(); err != nil {
		return val, apperrors.NewBadRequest("invalid form data", nil)
	}
	if err := getDecoder().Decode(&val, r.PostForm); err != nil {
		return val, apperrors.NewBadRequest("invalid form data", nil)
	}
	return val, nil
}
