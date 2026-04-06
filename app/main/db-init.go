package main

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/app"
	"github.com/healtronlabs/gofasta/app/graphql/resolvers"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/app/rest/routes"
	"github.com/healtronlabs/gofasta/app/services"
	"github.com/healtronlabs/gofasta/configs"
)

func setupAndInitializeDb(cfg *configs.AppConfig) (*handler.Server, *mux.Router) {
	db := configs.SetupDB(&cfg.Database)
	newUserService := services.NewUserService(db)
	graphqlResolver := handler.NewDefaultServer(app.NewExecutableSchema(app.Config{Resolvers: &resolvers.Resolver{
		UserService: newUserService,
	}}))
	apiRouter := routes.InitApiRoutes(&controllers.Controllers{
		UserController: controllers.NewUserControllerInstance(*newUserService),
	})
	return graphqlResolver, apiRouter
}
