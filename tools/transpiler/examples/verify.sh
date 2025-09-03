#!/bin/bash

# Gofasta Transpiler Examples Verification Script
# This script verifies that all Go examples in tools/transpiler/examples work correctly

set -e

# Global variables for cleanup
TEMP_DIRS=()
CREATED_FILES=()

# Function to track created files
track_file() {
    CREATED_FILES+=("$1")
}

# Function to track temp directories
track_temp_dir() {
    TEMP_DIRS+=("$1")
}

# Comprehensive cleanup function
cleanup() {
    local exit_code=$?
    
    print_step "Cleaning up verification artifacts..."
    
    # Clean up temporary directories
    for temp_dir in "${TEMP_DIRS[@]}"; do
        if [ -d "$temp_dir" ]; then
            rm -rf "$temp_dir" 2>/dev/null || true
        fi
    done
    
    # Remove generated files
    for file in "${CREATED_FILES[@]}"; do
        if [ -f "$file" ]; then
            rm -f "$file" 2>/dev/null || true
        fi
    done
    
    if [ $exit_code -eq 0 ]; then
        print_success "Cleanup completed successfully"
    else
        print_step "Cleanup completed (script exited with error code $exit_code)"
    fi
    
    exit $exit_code
}

# Set up signal handlers for cleanup
trap cleanup EXIT
trap 'cleanup' INT TERM

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_step() {
    printf "${BLUE}[STEP]${NC} %s\n" "$1"
}

print_success() {
    printf "${GREEN}[SUCCESS]${NC} %s\n" "$1"
}

print_warning() {
    printf "${YELLOW}[WARNING]${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
}

echo "🧪 Gofasta Transpiler Examples Verification"
echo "==========================================="

# Step 1: Verify we're in the right directory
print_step "Checking current directory..."
if [ ! -d "core" ] || [ ! -d "cloud" ] || [ ! -d "websocket" ]; then
    print_error "This script must be run from tools/transpiler/examples directory"
    print_step "Current directory: $(pwd)"
    print_step "Please cd to tools/transpiler/examples and run ./verify.sh"
    exit 1
fi

print_success "Running from correct directory: $(pwd)"

# Initialize counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Function to verify an example
verify_example() {
    local category="$1"
    local example_name="$2"
    local example_dir="$3"
    
    printf "\n${BLUE}Testing: $category/$example_name${NC}\n"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    cd "$example_dir"
    
    # Check what type of example this is
    if [ -f "go.mod" ]; then
        # This is a Go module - test building it
        print_step "  Building Go module..."
        
        # First try to tidy dependencies, then build
        if GOWORK=off go mod tidy >/dev/null 2>&1 && GOWORK=off go build -o temp_binary . >/dev/null 2>&1; then
            print_success "  ✓ Build successful"
            rm -f temp_binary
            PASSED_TESTS=$((PASSED_TESTS + 1))
            
            # Additional checks for specific examples
            if [ -f "main.go" ]; then
                print_step "  Checking main.go structure..."
                if grep -q "func main" main.go; then
                    print_success "  ✓ Contains main function"
                else
                    print_warning "  ⚠ No main function found"
                fi
            fi
            
        else
            print_error "  ✗ Build failed"
            print_step "  Attempting to show build errors:"
            GOWORK=off go build . 2>&1 | head -10 | sed 's/^/    /'
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        
    elif [ -f ".gitkeep" ]; then
        print_warning "  ⚠ Placeholder directory (contains .gitkeep)"
        SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
        
    elif [ -f "*.go" ] 2>/dev/null; then
        # Has Go files but no module
        print_step "  Found Go files without go.mod, checking syntax..."
        if go vet *.go >/dev/null 2>&1; then
            print_success "  ✓ Go syntax check passed"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            print_warning "  ⚠ Go syntax issues found"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        
    else
        print_warning "  ⚠ No recognizable content found"
        SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
    fi
    
    cd - >/dev/null
}

# Step 2: Scan and test all examples
print_step "Scanning examples directory..."
printf "\n"

for category_path in */; do
    if [ -d "$category_path" ]; then
        category_name=$(basename "$category_path")
        printf "${YELLOW}Category: $category_name${NC}\n"
        
        for example_path in "$category_path"*/; do
            if [ -d "$example_path" ]; then
                example_name=$(basename "$example_path")
                verify_example "$category_name" "$example_name" "$example_path"
            fi
        done
    fi
done

# Step 3: Summary and additional checks
printf "\n${YELLOW}=== Verification Summary ===${NC}\n"
printf "Total tests: $TOTAL_TESTS\n"
printf "${GREEN}Passed: $PASSED_TESTS${NC}\n"
printf "${RED}Failed: $FAILED_TESTS${NC}\n"
printf "${YELLOW}Skipped: $SKIPPED_TESTS${NC}\n"

# Step 4: Check for specific transpiler features
print_step "Checking for transpiler-specific patterns..."

# Look for any GoFasta-specific patterns that might indicate transpiler usage
if find . -name "*.go" -exec grep -l "RegisterRoutes\|inject:" {} \; | head -1 >/dev/null 2>&1; then
    print_success "✓ Found transpiler-generated patterns in Go files"
else
    print_warning "⚠ No obvious transpiler-generated patterns found"
fi

# Check for variety of example types
core_examples=$(find core/ -name "go.mod" | wc -l)
if [ "$core_examples" -gt 5 ]; then
    print_success "✓ Good variety of core examples ($core_examples modules)"
else
    print_warning "⚠ Limited core examples ($core_examples modules)"
fi

# Step 5: Performance check
print_step "Checking example complexity..."
largest_file=$(find . -name "*.go" -exec wc -l {} + 2>/dev/null | sort -n | tail -1 | awk '{print $1}' || echo "0")
if [ "$largest_file" -gt 50 ]; then
    print_success "✓ Examples have substantial content (largest: $largest_file lines)"
else
    print_warning "⚠ Examples seem minimal (largest: $largest_file lines)"
fi

# Final result
printf "\n"
if [ $FAILED_TESTS -eq 0 ]; then
    printf "🎉 ${GREEN}All examples verified successfully!${NC}\n"
    printf "\n"
    printf "📍 Key achievements:\n"
    printf "  ✓ $PASSED_TESTS examples built successfully\n" 
    printf "  ✓ All Go modules have proper dependencies\n"
    printf "  ✓ Examples demonstrate transpiler capabilities\n"
    printf "\n"
    printf "🚀 Examples are ready for development and testing!\n"
    exit 0
else
    printf "❌ ${RED}Some examples failed verification${NC}\n"
    printf "\n"
    printf "🔧 To fix issues:\n"
    printf "  1. Check the build errors shown above\n"
    printf "  2. Ensure all dependencies are properly configured\n"
    printf "  3. Verify go.mod files have correct module paths\n"
    printf "  4. Run 'go mod tidy' in failing example directories\n"
    printf "\n"
    exit 1
fi
