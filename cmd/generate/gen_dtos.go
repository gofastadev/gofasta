package generate

import (
	"fmt"

	"github.com/healtronlabs/gofasta/cmd/generate/templates"
)

func GenDTOs(d ScaffoldData) error {
	return WriteTemplate(fmt.Sprintf("app/dtos/%s.dtos.go", d.SnakeName), "dtos", templates.DTOs, d)
}
