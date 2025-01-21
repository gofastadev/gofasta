package main

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gorilla/mux"
	"github.com/healtronlabs/go_gql_template/app"
	"github.com/healtronlabs/go_gql_template/app/controllers"
	"github.com/healtronlabs/go_gql_template/app/resolvers"
	"github.com/healtronlabs/go_gql_template/app/routes"
	"github.com/healtronlabs/go_gql_template/app/services"
	"github.com/healtronlabs/go_gql_template/configs"
)

func setupAndInitializeDb() (*handler.Server, *mux.Router) {
	db := configs.SetupDB()
	newUserService := services.NewUserService(db)
	graphqlResolver := handler.NewDefaultServer(app.NewExecutableSchema(app.Config{Resolvers: &resolvers.Resolver{
		UserService: newUserService,
	}}))
	apiRouter := routes.InitApiRoutes(&controllers.Controllers{
		UserController: controllers.NewUserControllerInstance(*newUserService),
	})
	return graphqlResolver, apiRouter
}
