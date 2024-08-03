package goTypes

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id          uuid.UUID `json:"id"`
	FirstName   string    `json:"firstName"`
	OtherNames  string    `json:"otherNames"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phoneNumber"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type NewUserDto struct {
	FirstName   string `json:"firstName"`
	OtherNames  string `json:"otherNames"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type UserFiltersDto struct {
	FirstName   *string `json:"firstName,omitempty"`
	OtherNames  *string `json:"otherNames,omitempty"`
	Email       *string `json:"email,omitempty"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
}
