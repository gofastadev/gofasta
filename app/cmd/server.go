package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
)

func serve() {
	port := os.Getenv("PORT")
	playgroundRoute := os.Getenv("GRAPHQL_PLAYGROUND_ROUTE")
	graphqlGeneralRoute := os.Getenv("GRAPHQL_GENERAL_ROUTE")
	if port == "" {
		port = "8080"
	}
	if playgroundRoute == "" {
		playgroundRoute = "/graphql-playground"
	}
	if graphqlGeneralRoute == "" {
		graphqlGeneralRoute = "/graphql"
	}

	// Initialize routes
	graphqlResolver, apiRouter := setupAndInitializeDb()

	// Combine routes
	http.Handle(playgroundRoute, playground.Handler("GraphQL playground", graphqlGeneralRoute))
	http.Handle(graphqlGeneralRoute, graphqlResolver)
	http.Handle("/", apiRouter)

	log.Printf("connect to http://localhost:%s%s for GraphQL playground", port, playgroundRoute)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
