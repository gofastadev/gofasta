package services

import (
	"github.com/healtronlabs/go-gql-template/app/graphql/goTypes"
	"github.com/healtronlabs/go-gql-template/app/models"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{DB: db}
}

func (u *UserService) GetUsers() ([]*goTypes.User, error) {
	var res []*goTypes.User
	result := u.DB.Find(&res)
	if result.Error != nil {
		return nil, result.Error
	}
	return res, nil
}

func (u *UserService) CreateUser(input goTypes.NewUser) (*goTypes.User, error) {
	userData := models.User{Name: input.Name}
	result := u.DB.Create(&userData)
	user := &goTypes.User{Name: userData.Name, Id: userData.ID}
	return user, result.Error
}
