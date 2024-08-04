package dtos

type NewUserDto struct {
	FirstName   string `json:"firstName" validate:"required"`
	OtherNames  string `json:"otherNames" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	PhoneNumber string `json:"phoneNumber" validate:"required,len=10"`
}
