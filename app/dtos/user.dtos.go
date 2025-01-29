package dtos

import "github.com/google/uuid"

type NewUserDto struct {
	FirstName   string `json:"firstName" schema:"firstName" validate:"required"`
	OtherNames  string `json:"otherNames" schema:"otherNames" validate:"required"`
	Email       string `json:"email" schema:"email" validate:"required,email"`
	PhoneNumber string `json:"phoneNumber" schema:"phoneNumber" validate:"required,len=10"`
}

type TUserFieldsForUpdateDto struct {
	ID             uuid.UUID `json:"id" validate:"required,uuid4_valid,does_record_exist_by_id_for_verification=users"`
	RecordVersion  int       `json:"recordVersion" validate:"required,min=1"`
	FirstName      *string   `json:"firstName,omitempty"`
	OtherNames     *string   `json:"otherNames,omitempty"`
	Email          *string   `json:"email,omitempty" validate:"omitempty,is_record_exist_by_email_for_conflict=users"`
	PhoneNumber    *string   `json:"phoneNumber,omitempty" validate:"omitempty,is_record_exist_by_phone_number_for_conflict=users"`
	IsActive       *bool     `json:"isActive,omitempty"`
	IsDeletable    *bool     `json:"isDeletable,omitempty"`
}

type UserFieldsForFiltersDto struct {
	FirstName   *string `json:"firstName,omitempty" schema:"firstName"`
	OtherNames  *string `json:"otherNames,omitempty" schema:"otherNames"`
	Email       *string `json:"email,omitempty" schema:"email"`
	PhoneNumber *string `json:"phoneNumber,omitempty" schema:"phoneNumber"`
}

type UserFiltersDto struct {
	Fields     *UserFieldsForFiltersDto `json:"fields" schema:"fields"`
	Pagination *TPaginationInputDto     `json:"pagination,omitempty" schema:"pagination"`
	Sorting    *TSortingInputDto        `json:"sorting,omitempty" schema:"sorting"`
}

type TUserFiltersQueryParamsDto struct {
	FirstName       *string          `json:"firstName,omitempty" schema:"firstName"`
	OtherNames      *string          `json:"otherNames,omitempty" schema:"otherNames"`
	Email           *string          `json:"email,omitempty" schema:"email"`
	PhoneNumber     *string          `json:"phoneNumber,omitempty" schema:"phoneNumber"`
	Limit           *int             `json:"limit,omitempty" schema:"limit" validate:"gte=1"`
	Page            *int             `json:"page,omitempty" schema:"page" validate:"gte=1"`
	SortByField     string           `json:"sortByField" schema:"sortByField" validate:"required"`
	SortOrientation *SortOrientation `json:"sortOrientation,omitempty" schema:"sortOrientation" validate:"omitempty"`
}
