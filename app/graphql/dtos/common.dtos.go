package dtos

type TPaginationInputDto struct {
	Limit *int `json:"limit,omitempty" validate:"gte=1"`
	Page  *int `json:"page,omitempty" validate:"gte=1"`
}
