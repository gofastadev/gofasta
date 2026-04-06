package generate

import (
	"fmt"

	"github.com/healtronlabs/gofasta/cmd/generate/templates"
)

func GenMigration(d ScaffoldData) error {
	up := fmt.Sprintf("db/migrations/%s_create_%s.up.sql", d.MigrationNum, d.PluralSnake)
	down := fmt.Sprintf("db/migrations/%s_create_%s.down.sql", d.MigrationNum, d.PluralSnake)
	if err := WriteTemplate(up, "mig_up", templates.MigUp, d); err != nil {
		return err
	}
	return WriteTemplate(down, "mig_down", templates.MigDown, d)
}
