package types

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortOrientation_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		value    SortOrientation
		expected bool
	}{
		{"ASC is valid", SortOrientationAsc, true},
		{"DESC is valid", SortOrientationDesc, true},
		{"empty is invalid", SortOrientation(""), false},
		{"lowercase asc is invalid", SortOrientation("asc"), false},
		{"random string is invalid", SortOrientation("RANDOM"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.IsValid())
		})
	}
}

func TestSortOrientation_String(t *testing.T) {
	assert.Equal(t, "ASC", SortOrientationAsc.String())
	assert.Equal(t, "DESC", SortOrientationDesc.String())
}

func TestAllSortOrientation(t *testing.T) {
	assert.Len(t, AllSortOrientation, 2)
	assert.Contains(t, AllSortOrientation, SortOrientationAsc)
	assert.Contains(t, AllSortOrientation, SortOrientationDesc)
}

func TestSortOrientation_UnmarshalGQL(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		expected  SortOrientation
		expectErr bool
	}{
		{"valid ASC", "ASC", SortOrientationAsc, false},
		{"valid DESC", "DESC", SortOrientationDesc, false},
		{"invalid string", "INVALID", SortOrientation(""), true},
		{"non-string input", 123, SortOrientation(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var so SortOrientation
			err := so.UnmarshalGQL(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, so)
			}
		})
	}
}

func TestSortOrientation_MarshalGQL(t *testing.T) {
	tests := []struct {
		name     string
		value    SortOrientation
		expected string
	}{
		{"ASC", SortOrientationAsc, `"ASC"`},
		{"DESC", SortOrientationDesc, `"DESC"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.value.MarshalGQL(&buf)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestSortOrientation_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    SortOrientation
		expected string
	}{
		{"ASC", SortOrientationAsc, `"ASC"`},
		{"DESC", SortOrientationDesc, `"DESC"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.value.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(b))
		})
	}
}

func TestSortOrientation_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  SortOrientation
		expectErr bool
	}{
		{"valid ASC", `"ASC"`, SortOrientationAsc, false},
		{"valid DESC", `"DESC"`, SortOrientationDesc, false},
		{"invalid value", `"INVALID"`, SortOrientation(""), true},
		{"not a quoted string", `ASC`, SortOrientation(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var so SortOrientation
			err := so.UnmarshalJSON([]byte(tt.input))
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, so)
			}
		})
	}
}

func TestSortOrientation_JSONRoundTrip(t *testing.T) {
	original := SortOrientationAsc
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded SortOrientation
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestTPaginationInputDto_JSON(t *testing.T) {
	limit := 20
	page := 3
	dto := TPaginationInputDto{Limit: &limit, Page: &page}

	data, err := json.Marshal(dto)
	require.NoError(t, err)

	var decoded TPaginationInputDto
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.NotNil(t, decoded.Limit)
	require.NotNil(t, decoded.Page)
	assert.Equal(t, 20, *decoded.Limit)
	assert.Equal(t, 3, *decoded.Page)
}

func TestTPaginationInputDto_JSON_Omitempty(t *testing.T) {
	dto := TPaginationInputDto{}
	data, err := json.Marshal(dto)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(data))
}

func TestTSortingInputDto_JSON(t *testing.T) {
	orientation := SortOrientationDesc
	dto := TSortingInputDto{
		SortByField:     "createdAt",
		SortOrientation: &orientation,
	}

	data, err := json.Marshal(dto)
	require.NoError(t, err)

	var decoded TSortingInputDto
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "createdAt", decoded.SortByField)
	require.NotNil(t, decoded.SortOrientation)
	assert.Equal(t, SortOrientationDesc, *decoded.SortOrientation)
}

func TestTCommonResponseDto_JSON(t *testing.T) {
	msg := "success"
	fieldName := "email"
	dto := TCommonResponseDto{
		Status:  200,
		Message: &msg,
		Errors: []*TCommonAPIErrorDto{
			{FieldName: &fieldName, Message: "invalid email"},
		},
	}

	data, err := json.Marshal(dto)
	require.NoError(t, err)

	var decoded TCommonResponseDto
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, 200, decoded.Status)
	require.NotNil(t, decoded.Message)
	assert.Equal(t, "success", *decoded.Message)
	require.Len(t, decoded.Errors, 1)
	assert.Equal(t, "invalid email", decoded.Errors[0].Message)
}

func TestTPaginationObjectDto_JSON(t *testing.T) {
	total := 100
	perPage := 10
	totalPages := 10
	currentPage := 1
	dto := TPaginationObjectDto{
		TotalRecords:   &total,
		RecordsPerPage: &perPage,
		TotalPages:     &totalPages,
		CurrentPage:    &currentPage,
	}

	data, err := json.Marshal(dto)
	require.NoError(t, err)

	var decoded TPaginationObjectDto
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.NotNil(t, decoded.TotalRecords)
	assert.Equal(t, 100, *decoded.TotalRecords)
}
