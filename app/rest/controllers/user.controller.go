package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/healtronlabs/go_gql_template/app/dtos"
	"github.com/healtronlabs/go_gql_template/app/services"
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
func (uc *UserController) GetUsersWithFilters(w http.ResponseWriter, r *http.Request, filters dtos.TUserFiltersQueryParamsDto) {
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
	usersRes, err := uc.UserService.GetUsers(userFilters)
	if err != nil {
		http.Error(w, "Users not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usersRes)
}

// CreateUser handles POST /users requests.
func (uc *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user dtos.NewUserDto
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
	var dataForUpdate dtos.UserFieldsForUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&dataForUpdate); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dataForUpdate.ID = id
	updatedUser, err := uc.UserService.UpdateUser(dataForUpdate)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
}

// DeleteUser handles DELETE /users/{id} requests.
// func (uc *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	id, err := strconv.Atoi(vars["id"])
// 	if err != nil {
// 		http.Error(w, "Invalid user ID", http.StatusBadRequest)
// 		return
// 	}

// 	if err := uc.UserService.DeleteUser(id); err != nil {
// 		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusNoContent)
// }
