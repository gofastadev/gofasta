package generate

import "strings"

// ParseFields converts CLI args like "name:string price:float" into typed Fields.
func ParseFields(args []string) []Field {
	var fields []Field
	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			continue
		}
		f := Field{
			Name:      toPascalCase(parts[0]),
			JSONName:  toCamelCase(parts[0]),
			SnakeName: toSnakeCase(parts[0]),
		}
		switch strings.ToLower(parts[1]) {
		case "string":
			f.GoType, f.GormType, f.GQLType, f.SQLType = "string", `gorm:"not null"`, "String", "VARCHAR(255) NOT NULL"
		case "text":
			f.GoType, f.GormType, f.GQLType, f.SQLType = "string", `gorm:"type:text;not null"`, "String", "TEXT NOT NULL"
		case "int":
			f.GoType, f.GormType, f.GQLType, f.SQLType = "int", `gorm:"not null"`, "Int", "INTEGER NOT NULL"
		case "float":
			f.GoType, f.GormType, f.GQLType, f.SQLType = "float64", `gorm:"not null"`, "Float", "DECIMAL(10,2) NOT NULL"
		case "bool":
			f.GoType, f.GormType, f.GQLType, f.SQLType = "bool", `gorm:"not null;default:false"`, "Boolean", "BOOLEAN NOT NULL DEFAULT false"
		case "uuid":
			f.GoType, f.GormType, f.GQLType, f.SQLType = "uuid.UUID", `gorm:"type:uuid;not null"`, "ID", "UUID NOT NULL"
		case "time", "datetime":
			f.GoType, f.GormType, f.GQLType, f.SQLType = "time.Time", `gorm:"type:timestamp;not null"`, "DateTime", "TIMESTAMP NOT NULL DEFAULT now()"
		default:
			f.GoType, f.GormType, f.GQLType, f.SQLType = "string", `gorm:"not null"`, "String", "VARCHAR(255) NOT NULL"
		}
		fields = append(fields, f)
	}
	return fields
}
