package httputil

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
	apperrors "github.com/gofastadev/gofasta/pkg/errors"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
	decoder      *schema.Decoder
	decoderOnce  sync.Once
)

func getValidator() *validator.Validate {
	validateOnce.Do(func() { validate = validator.New() })
	return validate
}

func getDecoder() *schema.Decoder {
	decoderOnce.Do(func() {
		decoder = schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)
	})
	return decoder
}

// BindJSON decodes a JSON request body into T and validates it.
// Replaces manual json.NewDecoder + validate blocks with one line.
func BindJSON[T any](r *http.Request) (T, error) {
	var val T
	if err := json.NewDecoder(r.Body).Decode(&val); err != nil {
		return val, apperrors.NewBadRequest("invalid request payload", nil)
	}
	if err := getValidator().Struct(val); err != nil {
		return val, apperrors.NewValidation("validation failed", formatValidationErrors(err))
	}
	return val, nil
}

// BindQuery decodes URL query parameters into T and validates it.
func BindQuery[T any](r *http.Request) (T, error) {
	var val T
	if err := getDecoder().Decode(&val, r.URL.Query()); err != nil {
		return val, apperrors.NewBadRequest("invalid query parameters", nil)
	}
	if err := getValidator().Struct(val); err != nil {
		return val, apperrors.NewValidation("validation failed", formatValidationErrors(err))
	}
	return val, nil
}

// BindForm decodes form data into T and validates it.
func BindForm[T any](r *http.Request) (T, error) {
	var val T
	if err := r.ParseForm(); err != nil {
		return val, apperrors.NewBadRequest("invalid form data", nil)
	}
	if err := getDecoder().Decode(&val, r.PostForm); err != nil {
		return val, apperrors.NewBadRequest("invalid form data", nil)
	}
	if err := getValidator().Struct(val); err != nil {
		return val, apperrors.NewValidation("validation failed", formatValidationErrors(err))
	}
	return val, nil
}

func formatValidationErrors(err error) []map[string]string {
	var errs []map[string]string
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			errs = append(errs, map[string]string{
				"field":   fe.Field(),
				"message": fe.Error(),
			})
		}
	}
	return errs
}
