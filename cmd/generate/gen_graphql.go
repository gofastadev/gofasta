package generate

import (
	"fmt"

	"github.com/healtronlabs/gofasta/cmd/generate/templates"
)

func GenGraphQL(d ScaffoldData) error {
	return WriteTemplate(fmt.Sprintf("app/graphql/schema/%s.gql", d.SnakeName), "graphql", templates.GraphQL, d)
}
