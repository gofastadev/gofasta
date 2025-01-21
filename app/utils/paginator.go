package utils

import (
	"fmt"

	dtosGql "github.com/healtronlabs/go_gql_template/app/graphql/dtos_gql"
)

type PreparePaginating struct {
	PageFilters *dtosGql.TPaginationInputDto
	Sorting     *dtosGql.TSortingInputDto
}

func (p *PreparePaginating) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

func (p *PreparePaginating) GetLimit() int {
	if limit := p.PageFilters.Limit; limit != nil {
		return *limit
	}
	return 10
}

func (p *PreparePaginating) GetPage() int {
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
