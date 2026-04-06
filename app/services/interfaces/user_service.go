package interfaces

import (
	"context"

	"github.com/gofastadev/gofasta/app/dtos"
)

// UserServiceInterface defines the contract for user business logic.
type UserServiceInterface interface {
	FindUsersWithFilters(ctx context.Context, filters dtos.UserFiltersDto) (*dtos.TUsersResponseDto, error)
	CreateUser(ctx context.Context, input dtos.TCreateUserDto) (*dtos.TUserResponseDto, error)
	UpdateUser(ctx context.Context, input dtos.TUserFieldsForUpdateDto) (*dtos.TUserResponseDto, error)
	FindUserByID(ctx context.Context, filters dtos.TFindUserByIDDto) (*dtos.TUserResponseDto, error)
	ArchiveUser(ctx context.Context, input dtos.TArchiveUserDto) (*dtos.TCommonResponseDto, error)
}
