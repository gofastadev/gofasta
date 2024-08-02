package models

type User struct {
	BaseModelImpl
	Name string `json:"name" gorm:"not null"`
}

type NewUser struct {
	Name string `json:"name"`
}
