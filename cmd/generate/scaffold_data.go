package generate

import (
	"fmt"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// BuildScaffoldData converts a resource name and fields into fully computed ScaffoldData.
func BuildScaffoldData(name string, fields []Field) ScaffoldData {
	pascal := toPascalCase(name)
	plural := pluralize(pascal)
	driver := readDBDriver()

	// Resolve per-driver SQL type into the active SQLType field
	for i := range fields {
		fields[i].SQLType = resolveSQLType(fields[i], driver)
	}

	return ScaffoldData{
		Name:         pascal,
		LowerName:    toCamelCase(name),
		SnakeName:    toSnakeCase(name),
		PluralName:   plural,
		PluralSnake:  toSnakeCase(plural),
		PluralLower:  toCamelCase(plural),
		Fields:       fields,
		MigrationNum: nextMigrationNumber(),
		DBDriver:     driver,
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

// readDBDriver reads database.driver from config.yaml. Defaults to "postgres".
func readDBDriver() string {
	k := koanf.New(".")
	if _, err := os.Stat("config.yaml"); err == nil {
		k.Load(file.Provider("config.yaml"), yaml.Parser())
	}
	driver := k.String("database.driver")
	if driver == "" {
		return "postgres"
	}
	return driver
}

// resolveSQLType picks the correct SQL type string for the active database driver.
func resolveSQLType(f Field, driver string) string {
	switch driver {
	case "mysql":
		return f.SQLTypeMySQL
	case "sqlite":
		return f.SQLTypeSQLite
	case "sqlserver":
		return f.SQLTypeSQLServer
	case "clickhouse":
		return f.SQLTypeClickHouse
	default:
		return f.SQLTypePostgres
	}
}
