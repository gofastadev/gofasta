package tools

import(
	_ "github.com/99designs/gqlgen"
	_ "github.com/air-verse/air"
	_ "gorm.io/gorm"
	_ "gorm.io/driver/postgres"
	_ "github.com/google/uuid"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)
