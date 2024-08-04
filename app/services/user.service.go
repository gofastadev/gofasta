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
	totalPages := int(math.Ceil(float64(totalUsers) / float64(limit)))
	usersRes := query.Limit(paginator.GetLimit()).Offset(paginator.GetOffset()).Order(paginator.GetSort()).Find(&foundUsers)
	returnedRes := dtos.UsersResponseDto{
		Users: foundUsers,
		Pagination: &dtos.TPaginationObjectDto{
			TotalRecords:   &totalRecords,
			CurrentPage:    &page,
			RecordsPerPage: &limit,
			TotalPages:     &totalPages,
		},
	}
	return &returnedRes, usersRes.Error
}

func (u *UserService) CreateUser(input dtos.NewUserDto) (*dtos.UserResponseDto, error) {
	if validationErrors := utils.ValidateInput(input); len(validationErrors) > 0 {
		return &dtos.UserResponseDto{Errors: validationErrors}, nil
	}
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
	if err := u.DB.Create(&userData).Error; err != nil {
		return nil, err
	}
	user, err := castUserModelToUserDto(&userData)
	return &dtos.UserResponseDto{Data: user}, err
}

func (u *UserService) UpdateUser(input dtos.UserFieldsForUpdateDto) (*dtos.UserResponseDto, error) {
	if validationErrors := utils.ValidateInput(input); len(validationErrors) > 0 {
		return &dtos.UserResponseDto{Errors: validationErrors}, nil
	}
	userDataForUpdate := utils.ConvertStructToMap(input)
	if err := u.DB.Model(&models.User{}).Where("ID = ?", input.ID).Updates(userDataForUpdate).Error; err != nil {
		return nil, err
	}
	foundUser, err := u.findUserById(input.ID)
	return &dtos.UserResponseDto{Data: foundUser}, err
}

// PRIVATE FUNCTIONS
func (u *UserService) findUserById(id string) (*dtos.User, error) {
	if err := utils.ValidateIdStringIsValidUUID(id); err != nil {
		return nil, err
	}
	var user models.User
	if err := u.DB.Where("ID = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	foundUser, err := castUserModelToUserDto(&user)
	if err != nil {
		return nil, err
	}
	return foundUser, nil
}

func castUserModelToUserDto(user *models.User) (*dtos.User, error) {
	foundUser := &dtos.User{
		ID:          user.ID.String(),
		FirstName:   user.FirstName,
		OtherNames:  user.OtherNames,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
		CreatedAt:   user.CreatedAt.String(),
		UpdatedAt:   user.UpdatedAt.String(),
	}
	return foundUser, nil
}
