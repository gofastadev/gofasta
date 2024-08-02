package services

import (
	"log"

	"github.com/healtronlabs/go_gql_template/app/graphql/goTypes"
	"github.com/healtronlabs/go_gql_template/app/models"
	"github.com/healtronlabs/go_gql_template/app/utils"
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

func (u *UserService) CreateUser(input goTypes.NewUserDto) (*goTypes.User, error) {
	randomPassword, err := utils.GeneratePassword(16)
	if err != nil {
		log.Printf("Error while generating a random password: %v", err)
		return nil, err
	}
	userData := models.User{
		FirstName: input.FirstName,
		OtherNames: input.OtherNames,
		PhoneNumber: input.PhoneNumber,
		Email: input.Email,
		Password: randomPassword,
	}
	result := u.DB.Create(&userData)
	user := &goTypes.User{
		Id: userData.ID,
		FirstName: userData.FirstName,
		OtherNames: userData.OtherNames,
		PhoneNumber: userData.PhoneNumber,
		Email: userData.Email,
	}
	return user, result.Error
}
