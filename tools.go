//go:build tools

package tools

import (
	// CLI tools
	_ "github.com/99designs/gqlgen"
	_ "github.com/air-verse/air"
	_ "github.com/google/wire"
	_ "github.com/spf13/cobra"

	// Core dependencies
	_ "github.com/go-playground/validator/v10"
	_ "github.com/google/uuid"
	_ "github.com/gorilla/mux"
	_ "github.com/gorilla/schema"
	_ "gorm.io/driver/postgres"
	_ "gorm.io/gorm"

	// Config & logging
	_ "github.com/knadh/koanf/v2"
	_ "github.com/knadh/koanf/parsers/yaml"
	_ "github.com/knadh/koanf/providers/env"
	_ "github.com/knadh/koanf/providers/file"

	// Testing
	_ "github.com/stretchr/testify"
	_ "github.com/testcontainers/testcontainers-go"
	_ "github.com/testcontainers/testcontainers-go/modules/postgres"
)
