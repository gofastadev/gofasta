package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Create a new gofasta project",
	Long: `Bootstrap a new gofasta project from scratch. Creates the directory,
initializes Go modules, sets up the full project structure, and prepares
everything for development.

Examples:
  gofasta new myapp
  gofasta new github.com/myorg/myapp`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNew(args[0])
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}

func runNew(nameOrPath string) error {
	// Determine project name and module path
	projectName := filepath.Base(nameOrPath)
	modulePath := nameOrPath
	if !strings.Contains(modulePath, "/") {
		modulePath = projectName // simple name, module path = name
	}

	// Check if directory already exists
	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("directory %q already exists", projectName)
	}

	fmt.Printf("🚀 Creating new gofasta project: %s\n\n", projectName)

	// Step 1: Create project directory
	fmt.Printf("📁 Creating directory %s/\n", projectName)
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return err
	}

	// Change into the new directory
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectName); err != nil {
		return err
	}
	defer os.Chdir(origDir)

	// Step 2: Initialize go module
	fmt.Printf("📦 Initializing Go module: %s\n", modulePath)
	if err := runCmdIn("go", "mod", "init", modulePath); err != nil {
		return fmt.Errorf("go mod init failed: %w", err)
	}

	// Step 3: Create project structure
	fmt.Println("🏗  Creating project structure...")
	dirs := []string{
		"app/main",
		"app/models",
		"app/dtos",
		"app/services/interfaces",
		"app/repositories/interfaces",
		"app/rest/controllers",
		"app/rest/routes",
		"app/graphql/schema",
		"app/graphql/resolvers",
		"app/validators",
		"app/utils",
		"app/di/providers",
		"app/jobs",
		"configs",
		"cmd",
		"cmd/generate",
		"cmd/generate/templates",
		"db/migrations",
		"deployments/dev/app",
		"deployments/dev/db",
		"pkg/errors",
		"pkg/httputil",
		"pkg/logger",
		"pkg/middleware",
		"pkg/mailer",
		"pkg/scheduler",
		"templates/emails/layouts",
		"testutil/mocks",
		"testutil/testdb",
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	// Step 4: Create essential files
	fmt.Println("📝 Creating files...")

	writeProjectFile("app/main/main.go", fmt.Sprintf(`package main

import "%s/cmd"

func main() {
	cmd.Execute()
}
`, modulePath))

	writeProjectFile("config.yaml", `server:
  port: "8080"
  shutdown_timeout: 15s
  allowed_origins:
    - "*"

database:
  driver: postgres
  host: localhost
  port: "5432"
  sslmode: disable
  max_idle: 10
  max_open: 100
  max_life: 1h

graphql:
  playground_route: /graphql-playground
  general_route: /graphql

log:
  level: info
  format: text

email:
  provider: smtp
  from_name: `+projectName+`
  from_address: noreply@example.com
  smtp:
    host: localhost
    port: 587
    use_tls: true

jobs: []
`)

	writeProjectFile(".env.example", `# `+projectName+` environment config
PORT=8080
GOFASTA_DATABASE_DRIVER=postgres
GOFASTA_DATABASE_USER=`+projectName+`
GOFASTA_DATABASE_PASSWORD=`+projectName+`
GOFASTA_DATABASE_NAME=`+projectName+`_dev
DB_HOST_PORT=5433
GOFASTA_LOG_LEVEL=debug
`)

	writeProjectFile(".env", `PORT=8080
GOFASTA_DATABASE_USER=`+projectName+`
GOFASTA_DATABASE_PASSWORD=`+projectName+`
GOFASTA_DATABASE_NAME=`+projectName+`_dev
DB_HOST_PORT=5433
GOFASTA_LOG_LEVEL=debug
`)

	writeProjectFile(".gitignore", `# Binaries
tmp/
*.exe

# Environment
.env

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`)

	writeProjectFile(".go-version", "1.25.0\n")

	writeProjectFile("Makefile", `# `+projectName+` Makefile

.PHONY: dev up down build wire gqlgen test lint clean migrate-up migrate-down init

dev:
	go run ./app/main dev

up:
	docker compose up --build

down:
	docker compose down

build:
	go build -o ./tmp/main ./app/main/

wire:
	go tool wire ./app/di/

gqlgen:
	go tool gqlgen generate

test:
	go test ./... -short -v

lint:
	golangci-lint run

migrate-up:
	go run ./app/main migrate up

migrate-down:
	go run ./app/main migrate down

init:
	go run ./app/main init

clean:
	rm -rf ./tmp && go clean
`)

	writeProjectFile(".air.toml", `root = "."
tmp_dir = "tmp"

[build]
  bin = "./tmp/main serve"
  cmd = "go build -o ./tmp/main ./app/main/"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "yaml", "gql"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = true
`)

	writeProjectFile("compose.yaml", `services:
  app:
    build:
      context: .
      dockerfile: deployments/dev/app/dockerfile
    container_name: `+projectName+`_app
    ports:
      - "${PORT:-8080}:${PORT:-8080}"
    volumes:
      - .:/gofasta
      - go_modules:/go/pkg/mod
      - go_build_cache:/root/.cache/go-build
    environment:
      - PORT=${PORT:-8080}
      - GOFASTA_DATABASE_HOST=db
      - GOFASTA_DATABASE_PORT=5432
      - GOFASTA_DATABASE_USER=${GOFASTA_DATABASE_USER:-`+projectName+`}
      - GOFASTA_DATABASE_PASSWORD=${GOFASTA_DATABASE_PASSWORD:-`+projectName+`}
      - GOFASTA_DATABASE_NAME=${GOFASTA_DATABASE_NAME:-`+projectName+`_dev}
      - GOFASTA_DATABASE_SSLMODE=disable
      - GOFASTA_LOG_LEVEL=debug
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped

  db:
    image: postgres:16-alpine
    container_name: `+projectName+`_db
    ports:
      - "${DB_HOST_PORT:-5433}:5432"
    environment:
      POSTGRES_USER: ${GOFASTA_DATABASE_USER:-`+projectName+`}
      POSTGRES_PASSWORD: ${GOFASTA_DATABASE_PASSWORD:-`+projectName+`}
      POSTGRES_DB: ${GOFASTA_DATABASE_NAME:-`+projectName+`_dev}
    volumes:
      - db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${GOFASTA_DATABASE_USER:-`+projectName+`}"]
      interval: 2s
      timeout: 5s
      retries: 10
    restart: unless-stopped

volumes:
  db_data:
  go_modules:
  go_build_cache:
`)

	// Step 5: Install gofasta as dependency
	fmt.Println("\n📦 Installing gofasta framework...")
	if err := runCmdIn("go", "get", "github.com/healtronlabs/gofasta@latest"); err != nil {
		fmt.Println("   ⚠ Could not install gofasta (you may need to add it manually)")
	}

	// Step 6: Tidy
	fmt.Println("📦 Running go mod tidy...")
	runCmdIn("go", "mod", "tidy")

	// Step 7: Initialize git
	fmt.Println("\n🔧 Initializing git repository...")
	runCmdIn("git", "init")
	runCmdIn("git", "add", ".")
	runCmdIn("git", "commit", "-m", "Initial commit: gofasta project scaffold")

	fmt.Printf("\n✅ Project %s created successfully!\n", projectName)
	fmt.Printf("\nGet started:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  make up                        # Start with Docker (recommended)\n")
	fmt.Printf("  # or\n")
	fmt.Printf("  docker compose up db -d        # Start DB only\n")
	fmt.Printf("  make dev                       # Run on host with hot reload\n")
	fmt.Printf("\nGenerate resources:\n")
	fmt.Printf("  gofasta g s Product name:string price:float\n")
	return nil
}

func writeProjectFile(path string, content string) {
	dir := filepath.Dir(path)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}
	os.WriteFile(path, []byte(content), 0644)
	fmt.Printf("   %s\n", path)
}

func runCmdIn(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
