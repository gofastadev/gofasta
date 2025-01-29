package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/healtronlabs/gofasta/app/dtos"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/app/utils"
)

func UserRoutes(r *mux.Router, userController *controllers.UserController) {
	var decoder = schema.NewDecoder()
	// Base route for /users
	r.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		decoder.IgnoreUnknownKeys(true)
		switch r.Method {
		case http.MethodPost:
			userController.CreateUser(w, r)
		case http.MethodGet:
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse query parameters", http.StatusBadRequest)
				return
			}
			var filters dtos.TUserFiltersQueryParamsDto
			if err := decoder.Decode(&filters, r.URL.Query()); err != nil {
				http.Error(w, "Invalid query parameters: "+err.Error(), http.StatusBadRequest)
				return
			}

			userController.FindUsersWithFilters(w, r, filters)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodPost, http.MethodGet)

	// Nested route for /users/{id}
	r.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		passedId, err := utils.ParseIdStringIsValidUUID(id)
		if err != nil {
			http.Error(w, "id should be a valid UUID", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut:
			userController.UpdateUser(w, r, passedId)
		case http.MethodDelete:
			userController.ArchiveUser(w, r, passedId)
		case http.MethodGet:
			userController.FindUserById(w, r, passedId)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
}
