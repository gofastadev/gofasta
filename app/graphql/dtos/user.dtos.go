package dtos

type NewUserDto struct {
	FirstName   string `json:"firstName" validate:"required"`
	OtherNames  string `json:"otherNames" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	PhoneNumber string `json:"phoneNumber" validate:"required,len=10"`
}

type UserFieldsForUpdateDto struct {
	ID          string  `json:"id" validate:"uuidv4"`
	FirstName   *string `json:"firstName,omitempty"`
	OtherNames  *string `json:"otherNames,omitempty"`
	Email       *string `json:"email,omitempty"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
}
