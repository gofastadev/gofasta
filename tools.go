package tools

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/air-verse/air"
	_ "github.com/go-playground/validator/v10"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/db/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/google/uuid"
	_ "gorm.io/driver/postgres"
	_ "gorm.io/gorm"
)
