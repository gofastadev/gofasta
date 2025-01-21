package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/healtronlabs/go_gql_template/app/graphql/dtos"
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

// GetUser handles GET /users/{id} requests.
// func (uc *UserController) GetUser(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	id, err := strconv.Atoi(vars["id"])
// 	if err != nil {
// 		http.Error(w, "Invalid user ID", http.StatusBadRequest)
// 		return
// 	}

// 	user, err := uc.UserService.GetUserByID(id)
// 	if err != nil {
// 		http.Error(w, "User not found", http.StatusNotFound)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(user)
// }

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
func (uc *UserController) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// id, err := strconv.Atoi(vars["id"])
	// if err != nil {
	// 	http.Error(w, "Invalid user ID", http.StatusBadRequest)
	// 	return
	// }

	var user dtos.UserFieldsForUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	dataForUpdate := dtos.UserFieldsForUpdateDto{
		ID: vars["id"],
	}
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
