# Installation Guide

This guide covers installing Gofasta framework and CLI tools for your development environment.

## Prerequisites

Before installing Gofasta, ensure you have:

- **Go 1.22 or later** - [Download Go](https://golang.org/dl/)
- **Git** - For version control and package management
- **Database** (optional) - PostgreSQL, MongoDB, MySQL, or SQLite

### Verify Prerequisites

```bash
# Check Go version
go version

# Check Git
git --version
```

## Installation Methods

### Method 1: Using Go Install (Recommended)

Install the Gofasta CLI tool globally:

```bash
go install github.com/healtronlabs/gofasta/packages/cli@latest
```

Verify installation:

```bash
gofasta version
```

### Method 2: Build from Source

Clone and build the latest development version:

```bash
# Clone the repository
git clone https://github.com/healtronlabs/gofasta.git
cd gofasta

# Initialize workspace
go work sync

# Build CLI tool
cd packages/cli
go build -o gofasta

# Move to PATH (optional)
sudo mv gofasta /usr/local/bin/
```

### Method 3: Download Pre-built Binaries

Download pre-built binaries from the [releases page](https://github.com/healtronlabs/gofasta/releases):

```bash
# Download for your platform
curl -L https://github.com/healtronlabs/gofasta/releases/latest/download/gofasta-linux-amd64 -o gofasta

# Make executable
chmod +x gofasta

# Move to PATH
sudo mv gofasta /usr/local/bin/
```

## Creating Your First Project

Once installed, create a new Gofasta project:

```bash
# Create a new API project
gofasta new my-first-api

# Navigate to project
cd my-first-api

# Install dependencies
go mod tidy

# Start development server
gofasta dev
```

Your API will be available at `http://localhost:8080`.

## Framework Installation in Existing Projects

To add Gofasta to an existing Go project:

### 1. Initialize Go Module (if not already done)

```bash
go mod init your-project-name
```

### 2. Add Gofasta Packages

```bash
# Add core framework packages
go get github.com/healtronlabs/gofasta/packages/core@latest
go get github.com/healtronlabs/gofasta/packages/http@latest
go get github.com/healtronlabs/gofasta/packages/orm@latest

# Add optional packages as needed
go get github.com/healtronlabs/gofasta/packages/auth@latest
go get github.com/healtronlabs/gofasta/packages/validation@latest
```

### 3. Create Basic Application

```go
package main

import (
    "log"
  
    "github.com/healtronlabs/gofasta/packages/core"
    "github.com/healtronlabs/gofasta/packages/http"
)

func main() {
    app := core.CreateApp(&AppModule{})
    app.RegisterModule(http.NewHTTPModule(8080))
  
    log.Println("Starting server on port 8080...")
    if err := app.Listen(8080); err != nil {
        log.Fatal(err)
    }
}

type AppModule struct {
    *core.BaseModule
}

func (m *AppModule) Configure() {
    // Configure your application here
}
```

## Version Management

### Check Current Version

```bash
# Check CLI version
gofasta version

# Check framework version in your project
go list -m github.com/healtronlabs/gofasta/packages/core
```

### Update to Latest Version

#### Update CLI Tool

```bash
# Update to latest version
go install github.com/healtronlabs/gofasta/packages/cli@latest

# Or update to specific version
go install github.com/healtronlabs/gofasta/packages/cli@v1.2.0
```

#### Update Framework in Project

```bash
# Update all Gofasta packages to latest
go get -u github.com/healtronlabs/gofasta/packages/core@latest
go get -u github.com/healtronlabs/gofasta/packages/http@latest
go get -u github.com/healtronlabs/gofasta/packages/orm@latest

# Or update to specific version
go get github.com/healtronlabs/gofasta/packages/core@v1.2.0

# Clean up dependencies
go mod tidy
```

#### Update All Dependencies

```bash
# Update all dependencies in your project
go get -u ./...
go mod tidy
```

### Version Compatibility

Gofasta follows semantic versioning (SemVer):

- **Major versions** (v2.0.0) - Breaking changes
- **Minor versions** (v1.1.0) - New features, backward compatible
- **Patch versions** (v1.0.1) - Bug fixes, backward compatible

#### Compatibility Matrix

| Gofasta Version | Go Version | Status        |
| --------------- | ---------- | ------------- |
| v1.0.x          | 1.22+      | ✅ Supported  |
| v0.9.x          | 1.21+      | ⚠️ Legacy   |
| v0.8.x          | 1.20+      | ❌ Deprecated |

### Pinning Versions

To pin specific versions in your `go.mod`:

```go
module your-project

go 1.22

require (
    github.com/healtronlabs/gofasta/packages/core v1.0.0
    github.com/healtronlabs/gofasta/packages/http v1.0.0
    github.com/healtronlabs/gofasta/packages/orm v1.0.0
)
```

## Development Environment Setup

### IDE Configuration

#### VS Code

Install the Gofasta extension:

```bash
# Install from marketplace or
code --install-extension healtronlabs.gofasta
```

Add to your `.vscode/settings.json`:

```json
{
    "go.toolsEnvVars": {
        "GOFASTA_ENV": "development"
    },
    "gofasta.autoGenerate": true,
    "gofasta.liveReload": true
}
```

#### GoLand/IntelliJ

1. Install Go plugin
2. Configure Gofasta as external tool
3. Set up file watchers for hot reload

### Environment Variables

Set up your development environment:

```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export GOFASTA_ENV=development
export DATABASE_URL=postgresql://localhost:5432/myapp
export JWT_SECRET=your-development-secret
```

### Database Setup

#### PostgreSQL

```bash
# Install PostgreSQL
brew install postgresql  # macOS
sudo apt-get install postgresql  # Ubuntu

# Start service
brew services start postgresql  # macOS
sudo systemctl start postgresql  # Ubuntu

# Create database
createdb myapp
```

#### MongoDB

```bash
# Install MongoDB
brew install mongodb-community  # macOS
sudo apt-get install mongodb  # Ubuntu

# Start service
brew services start mongodb-community  # macOS
sudo systemctl start mongod  # Ubuntu
```

## Troubleshooting

### Common Installation Issues

#### Go Version Too Old

```
Error: package github.com/healtronlabs/gofasta/packages/core: go.mod requires go >= 1.22
```

**Solution**: Update Go to version 1.22 or later.

#### Module Not Found

```
Error: cannot find module github.com/healtronlabs/gofasta/packages/core
```

**Solution**:

```bash
# Clear module cache and retry
go clean -modcache
go mod download
```

#### Permission Denied

```
Error: permission denied: /usr/local/bin/gofasta
```

**Solution**: Use sudo or install to user directory:

```bash
# Install to user bin directory
mkdir -p ~/bin
go env -w GOBIN=~/bin
go install github.com/healtronlabs/gofasta/packages/cli@latest

# Add to PATH in your shell profile
echo 'export PATH=$HOME/bin:$PATH' >> ~/.bashrc
```

### Build Issues

#### CGO Errors

If you encounter CGO-related errors:

```bash
# Disable CGO for pure Go build
CGO_ENABLED=0 go build
```

#### Module Checksum Mismatch

```bash
# Clear module cache
go clean -modcache
go mod download
```

### Getting Help

If you encounter issues:

1. **Check Documentation**: [docs.gofasta.dev](https://docs.gofasta.dev)
2. **GitHub Issues**: [github.com/healtronlabs/gofasta/issues](https://github.com/healtronlabs/gofasta/issues)
3. **Community Forum**: [community.gofasta.dev](https://community.gofasta.dev)
4. **Discord**: [discord.gg/gofasta](https://discord.gg/gofasta)

## Next Steps

After successful installation:

1. **Read the [Quick Start Guide](quickstart.md)**
2. **Explore [Examples](../examples/)**
3. **Check out [Best Practices](best-practices.md)**
4. **Join the [Community](community.md)**

## Uninstallation

To remove Gofasta:

```bash
# Remove CLI tool
rm $(which gofasta)

# Remove from project
go mod edit -droprequire github.com/healtronlabs/gofasta/packages/core
go mod edit -droprequire github.com/healtronlabs/gofasta/packages/http
go mod edit -droprequire github.com/healtronlabs/gofasta/packages/orm
go mod tidy
```
