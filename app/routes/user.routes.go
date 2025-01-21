package routes

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/healtronlabs/go_gql_template/app/controllers"
)

func UserRoutes(r *mux.Router, userController *controllers.UserController) {
	// Base route for /users
	r.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			userController.CreateUser(w, r)
		// case http.MethodGet:
		// 	userController.GetUsers(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodPost, http.MethodGet)

	// Nested route for /users/{id}
	r.HandleFunc("/users/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		// case http.MethodGet:
		// 	userController.GetUserByID(w, r, id)
		case http.MethodPut:
			userController.UpdateUser(w, r, id)
		// case http.MethodDelete:
		// 	userController.DeleteUserByID(w, r, id)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
}
