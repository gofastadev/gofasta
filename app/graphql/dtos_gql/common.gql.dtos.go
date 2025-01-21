package dtosGql

type TPaginationInputDto struct {
	Limit *int `json:"limit,omitempty" schema:"limit" validate:"gte=1"`
	Page  *int `json:"page,omitempty" schema:"page" validate:"gte=1"`
}

type TSortingInputDto struct {
	SortByField     string           `json:"sortByField" schema:"sortByField" validate:"required"`
	SortOrientation *SortOrientation `json:"sortOrientation,omitempty" schema:"sortOrientation" validate:"omitempty"`
}
