package routes

import (
	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
)

func InitApiRoutes(controllers *controllers.Controllers) *mux.Router {
	r := mux.NewRouter()

	UserRoutes(r, controllers.UserController)

	return r
}
