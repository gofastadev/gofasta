package dtosGql

type NewUserDto struct {
	FirstName   string `json:"firstName" schema:"firstName" validate:"required"`
	OtherNames  string `json:"otherNames" schema:"otherNames" validate:"required"`
	Email       string `json:"email" schema:"email" validate:"required,email"`
	PhoneNumber string `json:"phoneNumber" schema:"phoneNumber" validate:"required,len=10"`
}

type UserFieldsForUpdateDto struct {
	ID          string  `json:"id" schema:"id" validate:"uuid4"`
	FirstName   *string `json:"firstName,omitempty" schema:"firstName"`
	OtherNames  *string `json:"otherNames,omitempty" schema:"otherNames"`
	Email       *string `json:"email,omitempty" schema:"email"`
	PhoneNumber *string `json:"phoneNumber,omitempty" schema:"phoneNumber"`
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
