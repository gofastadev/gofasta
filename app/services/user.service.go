package services

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/healtronlabs/gofasta/app/dtos"
	"github.com/healtronlabs/gofasta/app/models"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
	"github.com/healtronlabs/gofasta/app/utils"
	"github.com/healtronlabs/gofasta/app/validators"
	"gorm.io/gorm"
)

// Compile-time check that UserService implements UserServiceInterface.
var _ svcInterfaces.UserServiceInterface = (*UserService)(nil)

type UserService struct {
	DB *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{DB: db}
}

func (u *UserService) FindUsersWithFilters(ctx context.Context, filters dtos.UserFiltersDto) (*dtos.TUsersResponseDto, error) {
	query, err := utils.BuildQueryForAnyModel(u.DB.WithContext(ctx).Model(&models.User{}), utils.ConvertStructToMap(filters.Fields))
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
	returnedRes := dtos.TUsersResponseDto{
		Data: foundUsers,
		Pagination: &dtos.TPaginationObjectDto{
			TotalRecords:   &totalRecords,
			CurrentPage:    &page,
			RecordsPerPage: &limit,
			TotalPages:     &totalPages,
		},
	}
	return &returnedRes, usersRes.Error
}

func (u *UserService) CreateUser(ctx context.Context, input dtos.TCreateUserDto) (*dtos.TUserResponseDto, error) {
	if validationErrors := validators.ValidateInput(input, u.DB); len(validationErrors) > 0 {
		return &dtos.TUserResponseDto{Errors: validationErrors}, nil
	}
	randomPassword, err := utils.GeneratePassword(16)
	if err != nil {
		slog.Error("failed to generate random password", "error", err)
		return nil, err
	}
	userData := models.User{
		FirstName:   input.FirstName,
		OtherNames:  input.OtherNames,
		PhoneNumber: input.PhoneNumber,
		Email:       input.Email,
		Password:    randomPassword,
	}
	if err := u.DB.WithContext(ctx).Create(&userData).Error; err != nil {
		return nil, err
	}
	user, err := castUserModelToUserDto(&userData)
	return &dtos.TUserResponseDto{Data: user}, err
}

func (u *UserService) UpdateUser(ctx context.Context, input dtos.TUserFieldsForUpdateDto) (*dtos.TUserResponseDto, error) {
	validationErrors := validators.ValidateInput(input, u.DB)
	if len(validationErrors) > 0 {
		return &dtos.TUserResponseDto{Errors: validationErrors}, nil
	}
	if userToUpd, _ := u.findUserByIdAndRecordVersion(ctx, input.ID, input.RecordVersion); userToUpd == nil {
		fieldName := "recordVersion"
		return &dtos.TUserResponseDto{Errors: []*dtos.TCommonAPIErrorDto{{FieldName: &fieldName, Message: "The record version you passed is not matching"}}}, nil
	}
	userDataForUpdate := utils.ConvertStructToMap(input)
	if err := u.DB.WithContext(ctx).Model(&models.User{}).Where("ID = ?", input.ID).Updates(userDataForUpdate).Error; err != nil {
		return nil, err
	}
	foundUser, err := u.findUserById(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return &dtos.TUserResponseDto{Data: foundUser}, nil
}

func (u *UserService) FindUserByID(ctx context.Context, filters dtos.TFindUserByIDDto) (*dtos.TUserResponseDto, error) {
	if validationErrors := validators.ValidateInput(filters, u.DB); len(validationErrors) > 0 {
		return &dtos.TUserResponseDto{Errors: validationErrors}, nil
	}
	if user, err := u.findUserById(ctx, filters.UserID); err == nil {
		return &dtos.TUserResponseDto{Data: user}, nil
	} else {
		return nil, err
	}
}

func (u *UserService) ArchiveUser(ctx context.Context, input dtos.TArchiveUserDto) (*dtos.TCommonResponseDto, error) {
	if validationErrors := validators.ValidateInput(input, u.DB); len(validationErrors) > 0 {
		return &dtos.TCommonResponseDto{Errors: validationErrors}, nil
	}
	if err := u.DB.WithContext(ctx).Model(&models.User{}).Where("ID = ? AND is_deletable = ?", input.UserID, true).Updates(map[string]interface{}{"deleted_at": time.Now(), "is_active": false}).Error; err != nil {
		return nil, err
	}
	status := 200
	message := "Success"
	return &dtos.TCommonResponseDto{Status: status, Message: &message}, nil
}

// PRIVATE FUNCTIONS
func (u *UserService) findUserById(ctx context.Context, id uuid.UUID) (*dtos.User, error) {
	var user models.User
	if err := u.DB.WithContext(ctx).Where("ID = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	foundUser, err := castUserModelToUserDto(&user)
	if err != nil {
		return nil, err
	}
	return foundUser, nil
}

func (u *UserService) findUserByIdAndRecordVersion(ctx context.Context, id uuid.UUID, recordVersion int) (*models.User, error) {
	var user models.User
	if err := u.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL AND record_version = ?", id, recordVersion).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func castUserModelToUserDto(user *models.User) (*dtos.User, error) {
	foundUser := &dtos.User{
		ID:            user.ID,
		RecordVersion: user.RecordVersion,
		FirstName:     user.FirstName,
		OtherNames:    user.OtherNames,
		PhoneNumber:   user.PhoneNumber,
		Email:         user.Email,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		IsActive:      user.IsActive,
		IsDeletable:   user.IsDeletable,
		DeletedAt:     &user.DeletedAt,
	}
	return foundUser, nil
}
