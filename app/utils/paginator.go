package utils

import (
	"fmt"

	"github.com/gofastadev/gofasta/app/dtos"
)

type PreparePaginating struct {
	PageFilters *dtos.TPaginationInputDto
	Sorting     *dtos.TSortingInputDto
}

func (p *PreparePaginating) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

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
