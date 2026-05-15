// Package utils provides shared utility functions for gofasta applications.
package utils

import (
	"fmt"
	"regexp"

	"github.com/gofastadev/gofasta/pkg/types"
)

// safeIdentifier matches a plain SQL identifier — letters, digits, and
// underscores, starting with a letter or underscore. Used by GetSort to
// reject any sortField that could carry a SQL injection payload from a
// query-string parameter.
var safeIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// PreparePaginating bundles pagination + sorting input DTOs and exposes
// helpers for deriving GORM-friendly offset/limit/order values.
type PreparePaginating struct {
	PageFilters *types.TPaginationInputDto
	Sorting     *types.TSortingInputDto
}

// GetOffset returns the SQL OFFSET based on the current page and limit.
func (p *PreparePaginating) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

// GetLimit returns the page size, defaulting to 10 when none is provided.
func (p *PreparePaginating) GetLimit() int {
	defaultLimit := 10
	if p.PageFilters == nil {
		return defaultLimit
	}
	if limit := p.PageFilters.Limit; limit != nil {
		return *limit
	}
	return defaultLimit
}

// GetPage returns the current page number, defaulting to 1.
func (p *PreparePaginating) GetPage() int {
	defaultPage := 1
	if p.PageFilters == nil {
		return defaultPage
	}
	if page := p.PageFilters.Page; page != nil {
		return *page
	}
	return 1
}

// GetSort returns the SQL ORDER BY clause derived from the sorting
// input, defaulting to "created_at DESC" when no sorting is provided.
//
// Both the field name and the orientation are validated:
//   - The field name (after CamelToSnake) must be a plain identifier
//     ([a-zA-Z_][a-zA-Z0-9_]*). GORM's Order() accepts arbitrary SQL
//     expressions, so feeding raw query-string input here would let
//     callers inject SQL via `?sortByField=col;DROP+TABLE+x;--`.
//     Anything that doesn't match the identifier shape falls back to
//     "created_at" so the request still completes safely.
//   - The orientation is matched against the typed enum (SortOrientation.IsValid).
//     Anything else falls back to "DESC".
//
// This sanitization is the boundary between user input and the ORM —
// callers in repository code can pass the result directly to .Order().
func (p *PreparePaginating) GetSort() string {
	sortField := "createdAt"
	orientation := types.SortOrientationDesc
	if p.Sorting != nil {
		if p.Sorting.SortByField != "" {
			sortField = p.Sorting.SortByField
		}
		if p.Sorting.SortOrientation != nil && p.Sorting.SortOrientation.IsValid() {
			orientation = *p.Sorting.SortOrientation
		}
	}
	column := CamelToSnake(sortField)
	if !safeIdentifier.MatchString(column) {
		column = "created_at"
	}
	return fmt.Sprintf("%s %s", column, orientation.String())
}
