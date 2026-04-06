package generate

import (
	"fmt"
	"os"
)

// BuildScaffoldData converts a resource name and fields into fully computed ScaffoldData.
func BuildScaffoldData(name string, fields []Field) ScaffoldData {
	pascal := toPascalCase(name)
	plural := pluralize(pascal)
	return ScaffoldData{
		Name:         pascal,
		LowerName:    toCamelCase(name),
		SnakeName:    toSnakeCase(name),
		PluralName:   plural,
		PluralSnake:  toSnakeCase(plural),
		PluralLower:  toCamelCase(plural),
		Fields:       fields,
		MigrationNum: nextMigrationNumber(),
	}
}

func nextMigrationNumber() string {
	entries, _ := os.ReadDir("db/migrations")
	max := 0
	for _, e := range entries {
		if len(e.Name()) >= 6 {
			var num int
			fmt.Sscanf(e.Name()[:6], "%d", &num)
			if num > max {
				max = num
			}
		}
	}
	return fmt.Sprintf("%06d", max+1)
}
