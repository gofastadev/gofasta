package main

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/healtronlabs/go-gql-template/app"
	"github.com/healtronlabs/go-gql-template/app/resolvers"
	"github.com/healtronlabs/go-gql-template/app/services"
	"github.com/healtronlabs/go-gql-template/database"
)

func setupAndInitializeDb() *handler.Server {
	db := database.SetupDB()
	newUserService := services.NewUserService(db)
	srv := handler.NewDefaultServer(app.NewExecutableSchema(app.Config{Resolvers: &resolvers.Resolver{
		UserService: newUserService,
	}}))
	return srv
}
