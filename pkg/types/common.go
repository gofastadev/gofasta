// Package types provides common DTO types used across gofasta applications.
package types

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

type TPaginationInputDto struct {
	Limit *int `json:"limit,omitempty" schema:"limit" validate:"gte=1"`
	Page  *int `json:"page,omitempty" schema:"page" validate:"gte=1"`
}

type TSortingInputDto struct {
	SortByField     string           `json:"sortByField" schema:"sortByField" validate:"required"`
	SortOrientation *SortOrientation `json:"sortOrientation,omitempty" schema:"sortOrientation" validate:"omitempty"`
}

type TCommonAPIErrorDto struct {
	FieldName *string `json:"fieldName,omitempty"`
	Message   string  `json:"message"`
}

type TCommonResponseDto struct {
	Status  int                   `json:"status"`
	Message *string               `json:"message,omitempty"`
	Errors  []*TCommonAPIErrorDto `json:"errors,omitempty"`
}

type TPaginationObjectDto struct {
	TotalRecords   *int `json:"totalRecords,omitempty"`
	RecordsPerPage *int `json:"recordsPerPage,omitempty"`
	TotalPages     *int `json:"totalPages,omitempty"`
	CurrentPage    *int `json:"currentPage,omitempty"`
}

type SortOrientation string

const (
	SortOrientationAsc  SortOrientation = "ASC"
	SortOrientationDesc SortOrientation = "DESC"
)

var AllSortOrientation = []SortOrientation{
	SortOrientationAsc,
	SortOrientationDesc,
}

func (e SortOrientation) IsValid() bool {
	switch e {
	case SortOrientationAsc, SortOrientationDesc:
		return true
	}
	return false
}

func (e SortOrientation) String() string {
	return string(e)
}

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

func (e SortOrientation) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

func (e *SortOrientation) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	return e.UnmarshalGQL(s)
}

func (e SortOrientation) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e.MarshalGQL(&buf)
	return buf.Bytes(), nil
}
