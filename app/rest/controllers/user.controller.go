package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/app/dtos"
	"github.com/healtronlabs/gofasta/app/services"
	"github.com/healtronlabs/gofasta/app/utils"
)

// UserController handles RESTful requests for users.
type UserController struct {
	UserService services.UserService
}

// NewUserController creates a new UserController.
func NewUserControllerInstance(userService services.UserService) *UserController {
	return &UserController{UserService: userService}
}

// GetUser handles GET /users requests.
func (uc *UserController) FindUsersWithFilters(w http.ResponseWriter, r *http.Request, filters dtos.TUserFiltersQueryParamsDto) {
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
	usersRes, err := uc.UserService.FindUsersWithFilters(userFilters)
	if err != nil {
		http.Error(w, "Users not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usersRes)
}

// CreateUser handles POST /users requests.
func (uc *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user dtos.TCreateUserDto
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	createdUser, err := uc.UserService.CreateUser(user)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdUser)
}

// UpdateUser handles PUT /users/{id} requests.
func (uc *UserController) UpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	var dataForUpdate dtos.TUserFieldsForUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&dataForUpdate); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	userId, err := utils.ParseIdStringIsValidUUID(id)
	if err != nil {
		http.Error(w, "UserID should be a valid UUID", http.StatusBadRequest)
		return
	}
	dataForUpdate.ID = userId
	updatedUser, err := uc.UserService.UpdateUser(dataForUpdate)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
}

func (uc *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, err := utils.ParseIdStringIsValidUUID(vars["id"])
	if err != nil {
		http.Error(w, "UserID should be a valid UUID", http.StatusBadRequest)
		return
	}

	res, err := uc.UserService.ArchiveUser(dtos.TArchiveUserDto{UserID: userId})
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
