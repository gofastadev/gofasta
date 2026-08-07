package types

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// TPaginationInputDto is the shared pagination input DTO accepted by list
// endpoints. Both fields are optional pointers so absence can be distinguished
// from the zero value.
type TPaginationInputDto struct {
	Limit *int `json:"limit,omitempty" schema:"limit" validate:"gte=1"`
	Page  *int `json:"page,omitempty" schema:"page" validate:"gte=1"`
}

// TSortingInputDto is the shared sorting input DTO accepted by list endpoints.
type TSortingInputDto struct {
	SortByField     string           `json:"sortByField" schema:"sortByField" validate:"required"`
	SortOrientation *SortOrientation `json:"sortOrientation,omitempty" schema:"sortOrientation" validate:"omitempty"`
}

// TCommonAPIErrorDto represents a single field-level or global error returned
// by the framework's standard error response.
type TCommonAPIErrorDto struct {
	FieldName *string `json:"fieldName,omitempty"`
	Message   string  `json:"message"`
}

// TCommonResponseDto is the envelope returned by every framework-standard
// response, carrying an HTTP status, an optional message, and any errors.
type TCommonResponseDto struct {
	Status  int                   `json:"status"`
	Message *string               `json:"message,omitempty"`
	Errors  []*TCommonAPIErrorDto `json:"errors,omitempty"`
}

// TPaginationObjectDto describes the pagination metadata returned alongside a
// list response.
type TPaginationObjectDto struct {
	TotalRecords   *int `json:"totalRecords,omitempty"`
	RecordsPerPage *int `json:"recordsPerPage,omitempty"`
	TotalPages     *int `json:"totalPages,omitempty"`
	CurrentPage    *int `json:"currentPage,omitempty"`
}

// SortOrientation is the enum of supported sort directions (ASC/DESC).
type SortOrientation string

// Supported sort orientations.
const (
	SortOrientationAsc  SortOrientation = "ASC"
	SortOrientationDesc SortOrientation = "DESC"
)

// AllSortOrientation lists every valid SortOrientation value.
var AllSortOrientation = []SortOrientation{
	SortOrientationAsc,
	SortOrientationDesc,
}

// IsValid reports whether e is one of the supported SortOrientation values.
func (e SortOrientation) IsValid() bool {
	switch e {
	case SortOrientationAsc, SortOrientationDesc:
		return true
	}
	return false
}

// String returns the string form of the sort orientation.
func (e SortOrientation) String() string {
	return string(e)
}

// UnmarshalGQL implements gqlgen's Unmarshaler for SortOrientation.
func (e *SortOrientation) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SortOrientation(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SortOrientation", str)
	}
	return nil
}

// MarshalGQL implements gqlgen's Marshaler for SortOrientation.
func (e SortOrientation) MarshalGQL(w io.Writer) {
	_, _ = fmt.Fprint(w, strconv.Quote(e.String()))
}

// UnmarshalJSON implements json.Unmarshaler for SortOrientation.
func (e *SortOrientation) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	return e.UnmarshalGQL(s)
}

// MarshalJSON implements json.Marshaler for SortOrientation.
func (e SortOrientation) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e.MarshalGQL(&buf)
	return buf.Bytes(), nil
}
