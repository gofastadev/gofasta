package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
)

const defaultPort = "8080"

func serve() {

	port := os.Getenv("PORT")
	playgroundRoute := os.Getenv("GRAPHQL_PLAYGROUND_ROUTE")
	graphqlGeneralRoute := os.Getenv("GRAPHQL_GENERAL_ROUTE")
	if port == "" {
		port = defaultPort
	}
	if playgroundRoute == "" {
		playgroundRoute = "/graphql-playground"
	}
	if graphqlGeneralRoute == "" {
		graphqlGeneralRoute = "/graphql"
	}

	http.Handle(playgroundRoute, playground.Handler("GraphQL playground", graphqlGeneralRoute))
	http.Handle(graphqlGeneralRoute, setupAndInitializeDb())

	log.Printf("connect to http://localhost:%s%s for GraphQL playground", port, playgroundRoute)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
