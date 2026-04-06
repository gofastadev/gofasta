package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/healtronlabs/gofasta/app/dtos"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
	"github.com/healtronlabs/gofasta/app/utils"
	apperrors "github.com/healtronlabs/gofasta/pkg/errors"
	"github.com/healtronlabs/gofasta/pkg/httputil"
)

var decoder = schema.NewDecoder()

func init() {
	decoder.IgnoreUnknownKeys(true)
}

// UserController handles RESTful requests for users.
type UserController struct {
	UserService svcInterfaces.UserServiceInterface
}

// NewUserControllerInstance creates a new UserController.
func NewUserControllerInstance(userService svcInterfaces.UserServiceInterface) *UserController {
	return &UserController{UserService: userService}
}

// ListUsers handles GET /users requests.
func (uc *UserController) ListUsers(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return apperrors.NewBadRequest("failed to parse query parameters", nil)
	}
	var filters dtos.TUserFiltersQueryParamsDto
	if err := decoder.Decode(&filters, r.URL.Query()); err != nil {
		return apperrors.NewBadRequest("invalid query parameters", nil)
	}

	var userFilters dtos.UserFiltersDto
	userFilters.Fields = &dtos.UserFieldsForFiltersDto{
		FirstName:   filters.FirstName,
		OtherNames:  filters.OtherNames,
		Email:       filters.Email,
		PhoneNumber: filters.PhoneNumber,
	}
	userFilters.Pagination = &dtos.TPaginationInputDto{
		Limit: filters.Limit,
		Page:  filters.Page,
	}
	sortField := filters.SortByField
	if sortField == "" {
		sortField = "id"
	}
	userFilters.Sorting = &dtos.TSortingInputDto{
		SortByField:     sortField,
		SortOrientation: filters.SortOrientation,
	}

	usersRes, err := uc.UserService.FindUsersWithFilters(r.Context(), userFilters)
	if err != nil {
		return apperrors.NewInternal("failed to fetch users", err)
	}
	return httputil.OK(w, usersRes)
}

// CreateUser handles POST /users requests.
func (uc *UserController) CreateUser(w http.ResponseWriter, r *http.Request) error {
	var user dtos.TCreateUserDto
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return apperrors.NewBadRequest("invalid request payload", nil)
	}

	createdUser, err := uc.UserService.CreateUser(r.Context(), user)
	if err != nil {
		return apperrors.NewInternal("failed to create user", err)
	}
	return httputil.Created(w, createdUser)
}

// UpdateUser handles PUT /users/{id} requests.
func (uc *UserController) UpdateUser(w http.ResponseWriter, r *http.Request) error {
	userId, err := utils.ParseIdStringIsValidUUID(mux.Vars(r)["id"])
	if err != nil {
		return apperrors.NewBadRequest("id should be a valid UUID", nil)
	}

	var dataForUpdate dtos.TUserFieldsForUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&dataForUpdate); err != nil {
		return apperrors.NewBadRequest("invalid request payload", nil)
	}
	dataForUpdate.ID = userId

	updatedUser, err := uc.UserService.UpdateUser(r.Context(), dataForUpdate)
	if err != nil {
		return apperrors.NewInternal("failed to update user", err)
	}
	return httputil.OK(w, updatedUser)
}

// GetUser handles GET /users/{id} requests.
func (uc *UserController) GetUser(w http.ResponseWriter, r *http.Request) error {
	userId, err := utils.ParseIdStringIsValidUUID(mux.Vars(r)["id"])
	if err != nil {
		return apperrors.NewBadRequest("id should be a valid UUID", nil)
	}

	user, err := uc.UserService.FindUserByID(r.Context(), dtos.TFindUserByIDDto{UserID: userId})
	if err != nil {
		return apperrors.NewInternal("failed to find user", err)
	}
	return httputil.OK(w, user)
}

// ArchiveUser handles DELETE /users/{id} requests.
func (uc *UserController) ArchiveUser(w http.ResponseWriter, r *http.Request) error {
	userId, err := utils.ParseIdStringIsValidUUID(mux.Vars(r)["id"])
	if err != nil {
		return apperrors.NewBadRequest("id should be a valid UUID", nil)
	}

	res, err := uc.UserService.ArchiveUser(r.Context(), dtos.TArchiveUserDto{UserID: userId})
	if err != nil {
		return apperrors.NewInternal("failed to archive user", err)
	}
	return httputil.OK(w, res)
}
