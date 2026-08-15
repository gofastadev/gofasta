package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gofastadev/gofasta/pkg/types"
)

func TestSafeOrderBy_AllowedColumn(t *testing.T) {
	allowed := []string{"created_at", "email", "name"}

	got := SafeOrderBy("email", true, allowed, "created_at")
	assert.Equal(t, "email", got.Column.Name)
	assert.True(t, got.Desc)
}

func TestSafeOrderBy_CamelCaseIsSnakeCased(t *testing.T) {
	allowed := []string{"created_at", "first_name"}

	got := SafeOrderBy("firstName", false, allowed, "created_at")
	assert.Equal(t, "first_name", got.Column.Name)
	assert.False(t, got.Desc)
}

func TestSafeOrderBy_UnknownColumnFallsBack(t *testing.T) {
	allowed := []string{"created_at", "email"}

	// A column not in the allowlist must fall back, never reach SQL.
	got := SafeOrderBy("password", true, allowed, "created_at")
	assert.Equal(t, "created_at", got.Column.Name)
}

func TestSafeOrderBy_InjectionPayloadFallsBack(t *testing.T) {
	allowed := []string{"created_at", "email"}

	got := SafeOrderBy("email); DROP TABLE users;--", false, allowed, "created_at")
	assert.Equal(t, "created_at", got.Column.Name)
}

// --- PreparePaginating ---

func TestPreparePaginating_GetLimit(t *testing.T) {
	tests := []struct {
		name     string
		pager    PreparePaginating
		expected int
	}{
		{"nil PageFilters returns default 10", PreparePaginating{}, 10},
		{"nil Limit returns default 10", PreparePaginating{PageFilters: &types.TPaginationInputDto{}}, 10},
		{"custom limit", PreparePaginating{PageFilters: &types.TPaginationInputDto{Limit: intPtr(25)}}, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetLimit())
		})
	}
}

func TestPreparePaginating_GetPage(t *testing.T) {
	tests := []struct {
		name     string
		pager    PreparePaginating
		expected int
	}{
		{"nil PageFilters returns default 1", PreparePaginating{}, 1},
		{"nil Page returns default 1", PreparePaginating{PageFilters: &types.TPaginationInputDto{}}, 1},
		{"custom page", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(5)}}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetPage())
		})
	}
}

func TestPreparePaginating_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		pager    PreparePaginating
		expected int
	}{
		{"default page 1, limit 10 => offset 0", PreparePaginating{}, 0},
		{"page 2, limit 10 => offset 10", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(2), Limit: intPtr(10)}}, 10},
		{"page 3, limit 5 => offset 10", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(3), Limit: intPtr(5)}}, 10},
		{"page 1, limit 20 => offset 0", PreparePaginating{PageFilters: &types.TPaginationInputDto{Page: intPtr(1), Limit: intPtr(20)}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetOffset())
		})
	}
}

func TestPreparePaginating_GetSort(t *testing.T) {
	desc := types.SortOrientationDesc
	asc := types.SortOrientationAsc
	bogusOrientation := types.SortOrientation("DROP TABLE users")

	tests := []struct {
		name     string
		pager    PreparePaginating
		expected string
	}{
		{
			"nil sorting defaults to created_at DESC",
			PreparePaginating{},
			"created_at DESC",
		},
		{
			"empty sortByField defaults to created_at",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: ""}},
			"created_at DESC",
		},
		{
			"custom field and DESC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "updatedAt", SortOrientation: &desc}},
			"updated_at DESC",
		},
		{
			"custom field and ASC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "userName", SortOrientation: &asc}},
			"user_name ASC",
		},
		{
			"custom field, nil orientation defaults to DESC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "name"}},
			"name DESC",
		},
		// SQL-injection regression: any sortField that doesn't match a
		// plain identifier ([a-zA-Z_][a-zA-Z0-9_]*) is rejected and
		// replaced with "created_at" so the request still completes
		// safely. Without this, GORM's Order() would happily execute
		// arbitrary SQL embedded in the query string.
		{
			"injection via semicolon falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "id; DROP TABLE users;--"}},
			"created_at DESC",
		},
		{
			"injection via space falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "id ASC, name"}},
			"created_at DESC",
		},
		{
			"injection via quote falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: `id"--`}},
			"created_at DESC",
		},
		{
			"injection via leading digit falls back",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "1=1"}},
			"created_at DESC",
		},
		// Invalid orientation (e.g. unmarshaled from a malicious JSON
		// payload) falls back to DESC; we only honor the canonical
		// ASC/DESC enum values.
		{
			"invalid orientation falls back to DESC",
			PreparePaginating{Sorting: &types.TSortingInputDto{SortByField: "name", SortOrientation: &bogusOrientation}},
			"name DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pager.GetSort())
		})
	}
}

// --- helpers ---

func intPtr(v int) *int {
	return &v
}
