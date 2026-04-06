package generate

import (
	"fmt"

	"github.com/gofastadev/gofasta/cmd/generate/templates"
)

func GenRoutes(d ScaffoldData) error {
	return WriteTemplate(fmt.Sprintf("app/rest/routes/%s.routes.go", d.SnakeName), "routes", templates.Routes, d)
}
