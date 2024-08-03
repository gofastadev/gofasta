package services

import (
	"log"
	"math"

	"github.com/healtronlabs/go_gql_template/app/graphql/dtos"
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

func (u *UserService) GetUsers(filters dtos.UserFiltersDto) (*dtos.UsersResponseDto, error) {
	query, err := utils.BuildQueryForAnyModel(u.DB.Model(&models.User{}), utils.ConvertStructToMap(filters.Fields))
	if err != nil {
		return nil, err
	}
	var foundUsers []*dtos.User
	paginator := utils.PreparePaginating{PageFilters: filters.Pagination, Sorting: filters.Sorting}
	page := paginator.GetPage()
	limit := paginator.GetLimit()
	var totalUsers int64
	query.Count(&totalUsers)
	totalRecords := int(totalUsers)
	totalPages := int(math.Ceil(float64(totalUsers)/float64(limit)))
	usersRes := query.Limit(paginator.GetLimit()).Offset(paginator.GetOffset()).Order(paginator.GetSort()).Find(&foundUsers)
	returnedRes := dtos.UsersResponseDto{
		Users: foundUsers,
		Pagination: &dtos.TPaginationObjectDto{
			TotalRecords: &totalRecords,
			CurrentPage: &page,
			RecordsPerPage: &limit,
			TotalPages: &totalPages,
		},
	}
	return &returnedRes, usersRes.Error
}

func (u *UserService) CreateUser(input dtos.NewUserDto) (*dtos.User, error) {
	randomPassword, err := utils.GeneratePassword(16)
	if err != nil {
		log.Printf("Error while generating a random password: %v", err)
		return nil, err
	}
	userData := models.User{
		FirstName:   input.FirstName,
		OtherNames:  input.OtherNames,
		PhoneNumber: input.PhoneNumber,
		Email:       input.Email,
		Password:    randomPassword,
	}
	result := u.DB.Create(&userData)
	user := &dtos.User{
		ID:          userData.ID.String(),
		FirstName:   userData.FirstName,
		OtherNames:  userData.OtherNames,
		PhoneNumber: userData.PhoneNumber,
		Email:       userData.Email,
		CreatedAt: userData.CreatedAt.String(),
		UpdatedAt: userData.UpdatedAt.String(),
	}
	return user, result.Error
}
