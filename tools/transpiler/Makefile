# GoFasta Transpiler Makefile
# High-performance enterprise backend framework

.PHONY: all build test clean install dev lint coverage help run example
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := gofasta
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0-dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"
BUILD_FLAGS := -trimpath
RACE_FLAGS := -race
COVERAGE_FLAGS := -coverprofile=coverage.out -covermode=atomic

# Directories
BUILD_DIR := ./build
CACHE_DIR := ./.gofasta-cache
EXAMPLES_DIR := ./examples/basic
DIST_DIR := ./dist

##@ General

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

all: clean build test ## Build everything and run tests

##@ Build

build: ## Build the GoFasta transpiler
	@echo "🔨 Building GoFasta transpiler $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd
	@echo "✅ Built $(BUILD_DIR)/$(BINARY_NAME)"

build-race: ## Build with race detection
	@echo "🔨 Building GoFasta transpiler with race detection..."
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) $(RACE_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-race ./cmd
	@echo "✅ Built $(BUILD_DIR)/$(BINARY_NAME)-race"

build-all: ## Build for multiple platforms
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd
	@echo "✅ Built binaries in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

install: build ## Install the binary to GOPATH/bin
	@echo "📦 Installing $(BINARY_NAME)..."
	go install $(BUILD_FLAGS) $(LDFLAGS) ./cmd
	@echo "✅ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

##@ Development

dev: ## Build and run in development mode
	@echo "🚀 Running GoFasta in development mode..."
	go run $(BUILD_FLAGS) ./cmd -verbose

watch: ## Watch for changes and rebuild
	@echo "👁️  Watching for changes..."
	@command -v entr >/dev/null 2>&1 || (echo "❌ entr not found. Install with: brew install entr" && exit 1)
	find . -name "*.go" | entr -r make build

run: build ## Build and run with example files
	@echo "🚀 Running GoFasta transpiler..."
	$(BUILD_DIR)/$(BINARY_NAME) -input $(EXAMPLES_DIR) -output ./generated -verbose -force

example: ## Create and transpile example files
	@echo "📝 Creating example .gofa files..."
	@mkdir -p $(EXAMPLES_DIR)
	@$(MAKE) create-examples
	@echo "🚀 Transpiling examples..."
	$(BUILD_DIR)/$(BINARY_NAME) -input $(EXAMPLES_DIR) -output ./generated -verbose -force
	@echo "✅ Check ./generated/ for transpiled files"

dry-run: build ## Show what would be transpiled without doing it
	$(BUILD_DIR)/$(BINARY_NAME) -input $(EXAMPLES_DIR) -dry-run -verbose

##@ Testing

test: ## Run all tests
	@echo "🧪 Running tests..."
	go test -v ./...

test-race: ## Run tests with race detection
	@echo "🧪 Running tests with race detection..."
	go test -v $(RACE_FLAGS) ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	go test -v $(COVERAGE_FLAGS) ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

test-core: ## Run core package tests only
	@echo "🧪 Running core package tests..."
	go test -v ./core

test-cmd: ## Run CLI tests only
	@echo "🧪 Running CLI tests..."
	go test -v ./cmd

benchmark: ## Run benchmarks
	@echo "📊 Running benchmarks..."
	go test -v -bench=. -benchmem ./...

##@ Quality

lint: ## Run linter
	@echo "🔍 Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "❌ golangci-lint not found. Install with: brew install golangci-lint" && exit 1)
	golangci-lint run

fmt: ## Format code
	@echo "🎨 Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...

tidy: ## Tidy go modules
	@echo "🧹 Tidying modules..."
	go mod tidy

check: lint vet test ## Run all quality checks

##@ Utilities

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(DIST_DIR) ./generated $(CACHE_DIR) coverage.out coverage.html

deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	go mod download

version: ## Show version information
	@echo "GoFasta Transpiler"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

size: build ## Show binary size
	@echo "📏 Binary size:"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME) | awk '{print $$5 " " $$9}'

##@ Examples

create-examples: ## Create sample .gofa files for testing
	@mkdir -p $(EXAMPLES_DIR)
	@cat > $(EXAMPLES_DIR)/hello.gofa << 'EOF'
package main

import "fmt"

@Controller("/api/hello")
func HelloController() {
	fmt.Println("Hello from GoFasta!")
}

@Get("/")
func HelloWorld() string {
	return "Hello, World!"
}

func main() {
	HelloController()
	result := HelloWorld()
	fmt.Println(result)
}
EOF

	@cat > $(EXAMPLES_DIR)/user.controller.gofa << 'EOF'
package controllers

import (
	"fmt"
	"encoding/json"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
}

@Controller("/api/users")
@UseGuards("auth")
type UserController struct {
	users []User
}

@Get("/")
func (c *UserController) GetUsers() []User {
	return c.users
}

@Get("/:id")
func (c *UserController) GetUser(@Param("id") id string) *User {
	fmt.Printf("Getting user with ID: %s\n", id)
	// Mock implementation
	return &User{ID: 1, Name: "John Doe", Email: "john@example.com"}
}

@Post("/")
func (c *UserController) CreateUser(@Body() user User) User {
	fmt.Printf("Creating user: %+v\n", user)
	user.ID = len(c.users) + 1
	c.users = append(c.users, user)
	return user
}

@Put("/:id")
@Roles("admin", "editor")
func (c *UserController) UpdateUser(@Param("id") id string, @Body() user User) User {
	fmt.Printf("Updating user %s: %+v\n", id, user)
	return user
}

@Delete("/:id")
@Roles("admin")
func (c *UserController) DeleteUser(@Param("id") id string) map[string]string {
	fmt.Printf("Deleting user with ID: %s\n", id)
	return map[string]string{"message": "User deleted successfully"}
}
EOF

	@cat > $(EXAMPLES_DIR)/service.gofa << 'EOF'
package services

import (
	"fmt"
	"time"
)

@Injectable()
@Singleton()
type UserService struct {
	cache map[string]interface{}
}

@Async()
func (s *UserService) ProcessUser(id string) error {
	fmt.Printf("Processing user %s asynchronously...\n", id)
	time.Sleep(100 * time.Millisecond) // Simulate work
	return nil
}

@Cache(strategy: "redis", ttl: "1h")
func (s *UserService) GetUserProfile(id string) interface{} {
	fmt.Printf("Fetching user profile for %s\n", id)
	return map[string]interface{}{
		"id": id,
		"profile": "cached_profile_data",
		"lastAccessed": time.Now(),
	}
}

@Retry(maxAttempts: 3, delay: "1s")
@CircuitBreaker(threshold: 5, timeout: "30s")
func (s *UserService) ExternalAPICall(endpoint string) ([]byte, error) {
	fmt.Printf("Making external API call to %s\n", endpoint)
	// Mock implementation
	return []byte(`{"status": "success"}`), nil
}

@HealthCheck(interval: "30s")
func (s *UserService) CheckHealth() bool {
	return len(s.cache) < 1000 // Mock health check
}
EOF

	@echo "✅ Created example .gofa files in $(EXAMPLES_DIR)/"

clean-examples: ## Clean generated example files
	rm -rf $(EXAMPLES_DIR) ./generated

##@ Performance

profile-cpu: build ## Run CPU profiling
	@echo "📊 Running CPU profiling..."
	$(BUILD_DIR)/$(BINARY_NAME) -input $(EXAMPLES_DIR) -output ./generated -cpuprofile=cpu.prof -force -verbose
	go tool pprof cpu.prof

profile-mem: build ## Run memory profiling  
	@echo "📊 Running memory profiling..."
	$(BUILD_DIR)/$(BINARY_NAME) -input $(EXAMPLES_DIR) -output ./generated -memprofile=mem.prof -force -verbose
	go tool pprof mem.prof

stress-test: build create-examples ## Run stress test with many files
	@echo "🔥 Running stress test..."
	@mkdir -p ./stress-test
	@for i in {1..50}; do \
		cp $(EXAMPLES_DIR)/user.controller.gofa ./stress-test/user$$i.controller.gofa; \
	done
	@echo "⏱️  Transpiling 50 files..."
	time $(BUILD_DIR)/$(BINARY_NAME) -input ./stress-test -output ./generated-stress -verbose -force -parallel 8
	@rm -rf ./stress-test ./generated-stress
	@echo "✅ Stress test completed"

##@ Release

release-check: test lint ## Pre-release checks
	@echo "🔍 Running pre-release checks..."
	@echo "✅ All checks passed - ready for release"

package: build-all ## Package for distribution
	@echo "📦 Creating distribution packages..."
	@mkdir -p $(DIST_DIR)
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-linux-amd64
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-darwin-amd64  
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-darwin-arm64
	zip -j $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo "✅ Distribution packages created in $(DIST_DIR)/"

##@ Info

.PHONY: info
info: ## Show project information
	@echo "🚀 GoFasta Transpiler - Enterprise Backend Framework"
	@echo "=================================================="
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo ""
	@echo "📊 Project Stats:"
	@find . -name "*.go" -not -path "./vendor/*" | wc -l | xargs printf "Go files: %s\n"
	@find . -name "*.go" -not -path "./vendor/*" -exec wc -l {} + | tail -1 | awk '{print "Lines of code: " $$1}'
	@go list -m all | wc -l | xargs printf "Dependencies: %s\n"
	@echo ""
	@echo "🏗️  Core Components:"
	@ls -la ./core/*.go | wc -l | xargs printf "Core modules: %s\n"
	@ls -la ./examples/core/ | grep -v total | wc -l | xargs printf "Examples: %s\n"
	@find . -name "*_test.go" | wc -l | xargs printf "Test files: %s\n"