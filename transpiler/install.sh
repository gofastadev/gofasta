#!/bin/bash
# Gofasta Transpiler Installation Script

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

BINARY_NAME="gofasta"
INSTALL_DIR="${GOPATH:-$HOME/go}/bin"

print_banner() {
    echo -e "${BLUE}"
    echo "📦 Gofasta Transpiler Installer"
    echo "==============================="
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

check_requirements() {
    print_step "Checking requirements..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go first."
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "  ✓ Go version: ${GO_VERSION}"
    
    if [ -z "${GOPATH}" ]; then
        GOPATH=$(go env GOPATH)
        if [ -z "${GOPATH}" ]; then
            print_error "GOPATH is not set"
            exit 1
        fi
    fi
    
    echo "  ✓ GOPATH: ${GOPATH}"
    echo "  ✓ Install directory: ${INSTALL_DIR}"
    
    # Create install directory if it doesn't exist
    mkdir -p "${INSTALL_DIR}"
    
    print_success "Requirements check passed"
}

install_transpiler() {
    print_step "Installing Gofasta transpiler..."
    
    # Install using go install
    echo "  Installing from source..."
    go install -ldflags "-X main.version=installed" ./cmd
    
    # Verify installation
    if command -v "${BINARY_NAME}" &> /dev/null; then
        INSTALLED_PATH=$(which "${BINARY_NAME}")
        print_success "Installed successfully to: ${INSTALLED_PATH}"
        
        # Test the installation
        echo "  Testing installation..."
        "${BINARY_NAME}" -version
        
    else
        print_error "Installation failed. Binary not found in PATH."
        echo "  Make sure ${INSTALL_DIR} is in your PATH"
        exit 1
    fi
}

show_post_install() {
    echo ""
    echo -e "${GREEN}🎉 Installation completed successfully!${NC}"
    echo ""
    echo "To get started:"
    echo "  1. Create a .gofa file:"
    echo "     echo 'package main' > hello.gofa"
    echo "     echo '@Controller(\"/hello\")' >> hello.gofa"
    echo "     echo 'func Hello() { println(\"Hello Gofasta!\") }' >> hello.gofa"
    echo ""
    echo "  2. Transpile it:"
    echo "     ${BINARY_NAME} -input . -verbose"
    echo ""
    echo "  3. Get help:"
    echo "     ${BINARY_NAME} -help"
    echo ""
    echo "Documentation: https://github.com/healtronlabs/gofasta"
}

main() {
    print_banner
    check_requirements
    install_transpiler
    show_post_install
}

main "$@"