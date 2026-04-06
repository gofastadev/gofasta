// Gofasta API
//
//	@title			Gofasta API
//	@version		1.0
//	@description	A production-grade Go web framework with REST + GraphQL, DI, and full-stack features.
//	@host			localhost:8080
//	@BasePath		/api/v1
//	@schemes		http https
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter "Bearer {token}"
package main

import "github.com/healtronlabs/gofasta/cmd"

func main() {
	cmd.Execute()
}
