package resolvers

import "github.com/healtronlabs/go_gql_template/app/services"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	UserService *services.UserService
}
