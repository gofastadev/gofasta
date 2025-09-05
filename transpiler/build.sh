#!/bin/bash
# Gofasta Transpiler Build Script
# High-performance enterprise backend framework

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build configuration
BINARY_NAME="gofasta"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0-dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DIR="./dist"

# Go build flags
LDFLAGS="-ldflags=-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"
BUILD_FLAGS="-trimpath"

print_banner() {
    echo -e "${BLUE}"
    echo "🚀 Gofasta Transpiler Build Script"
    echo "===================================="
    echo "Version: ${VERSION}"
    echo "Build Time: ${BUILD_TIME}"
    echo "Git Commit: ${GIT_COMMIT:0:8}"
    echo -e "${NC}"
}

print_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    print_step "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "  ✓ Go version: ${GO_VERSION}"
    
    print_success "Dependencies check passed"
}

clean_build() {
    print_step "Cleaning previous build artifacts..."
    rm -rf "${BUILD_DIR}"
    rm -rf .gofasta-cache
    rm -f coverage.out coverage.html
    rm -rf ./generated
    print_success "Cleanup completed"
}

run_tests() {
    print_step "Running tests..."
    
    # Run core tests
    echo "  Running core package tests..."
    go test -v ./core
    
    # Run CLI tests  
    echo "  Running CLI tests..."
    go test -v ./cmd
    
    print_success "All tests passed"
}

build_binary() {
    print_step "Building Gofasta transpiler..."
    
    mkdir -p "${BUILD_DIR}"
    
    echo "  Building with version info..."
    go build -trimpath -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" -o "${BUILD_DIR}/${BINARY_NAME}" ./cmd
    
    # Check if binary was created
    if [ -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
        BINARY_SIZE=$(ls -lh "${BUILD_DIR}/${BINARY_NAME}" | awk '{print $5}')
        print_success "Binary built successfully: ${BUILD_DIR}/${BINARY_NAME} (${BINARY_SIZE})"
    else
        print_error "Binary build failed"
        exit 1
    fi
}

create_examples() {
    print_step "Creating example .gofa files..."
    
    EXAMPLES_DIR="./examples/basic"
    mkdir -p "${EXAMPLES_DIR}"
    
    # Create hello.gofa
    cat > "${EXAMPLES_DIR}/hello.gofa" << 'EOF'
package main

import "fmt"

@Controller("/api/hello")
func HelloController() {
	fmt.Println("Hello from Gofasta!")
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

    # Create user.controller.gofa
    cat > "${EXAMPLES_DIR}/user.controller.gofa" << 'EOF'
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
	return &User{ID: 1, Name: "John Doe", Email: "john@example.com"}
}

@Post("/")
func (c *UserController) CreateUser(@Body() user User) User {
	fmt.Printf("Creating user: %+v\n", user)
	user.ID = len(c.users) + 1
	c.users = append(c.users, user)
	return user
}
EOF

    # Create service.gofa
    cat > "${EXAMPLES_DIR}/service.gofa" << 'EOF'
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
	time.Sleep(100 * time.Millisecond)
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
EOF

    print_success "Created example files in ${EXAMPLES_DIR}/"
}

test_transpilation() {
    print_step "Testing transpilation pipeline..."
    
    EXAMPLES_DIR="./examples/basic"
    OUTPUT_DIR="./generated"
    
    if [ ! -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
        print_error "Binary not found. Please build first."
        exit 1
    fi
    
    # Test dry run
    echo "  Testing dry run mode..."
    "${BUILD_DIR}/${BINARY_NAME}" -input "${EXAMPLES_DIR}" -dry-run -verbose
    
    # Test actual transpilation
    echo "  Testing actual transpilation..."
    "${BUILD_DIR}/${BINARY_NAME}" -input "${EXAMPLES_DIR}" -output "${OUTPUT_DIR}" -verbose -force
    
    # Check generated files
    GENERATED_COUNT=$(find "${OUTPUT_DIR}" -name "*.go" 2>/dev/null | wc -l)
    if [ "${GENERATED_COUNT}" -gt 0 ]; then
        print_success "Generated ${GENERATED_COUNT} Go files"
        echo "  Generated files:"
        find "${OUTPUT_DIR}" -name "*.go" | sed 's/^/    /'
    else
        print_error "No files were generated"
        exit 1
    fi
}

show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  build       Build the Gofasta transpiler (default)"
    echo "  test        Run tests only"
    echo "  clean       Clean build artifacts"
    echo "  examples    Create example files only"
    echo "  transpile   Test transpilation pipeline"
    echo "  all         Clean, test, build, create examples, and test transpilation"
    echo "  help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0              # Build the transpiler"
    echo "  $0 all          # Full build and test pipeline"
    echo "  $0 test         # Run tests only"
    echo "  $0 clean        # Clean artifacts"
}

main() {
    print_banner
    
    command=${1:-build}
    
    case "$command" in
        "build")
            check_dependencies
            build_binary
            ;;
        "test")
            check_dependencies
            run_tests
            ;;
        "clean")
            clean_build
            ;;
        "examples")
            create_examples
            ;;
        "transpile")
            check_dependencies
            create_examples
            test_transpilation
            ;;
        "all")
            check_dependencies
            clean_build
            run_tests
            build_binary
            create_examples
            test_transpilation
            print_success "🎉 Full build pipeline completed successfully!"
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            print_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"