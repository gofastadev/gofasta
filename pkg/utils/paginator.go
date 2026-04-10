// Package utils provides shared utility functions for gofasta applications.
package utils

import (
	"fmt"

	"github.com/gofastadev/gofasta/pkg/types"
)

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

// GetSort returns the SQL ORDER BY clause derived from the sorting input,
// defaulting to "created_at DESC" when no sorting is provided.
func (p *PreparePaginating) GetSort() string {
	sortOrientation := "DESC"
	sortField := "createdAt"
	if p.Sorting != nil {
		sortField = p.Sorting.SortByField
		if p.Sorting.SortOrientation != nil {
			sortOrientation = p.Sorting.SortOrientation.String()
		}
	}
	return fmt.Sprintf("%s %s", CamelToSnake(sortField), sortOrientation)
}
