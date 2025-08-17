# Gofasta CLI

The official command-line interface for the Gofasta framework.

## Installation

```bash
go install github.com/healtronlabs/gofasta/packages/cli@latest
```

Or build from source:

```bash
git clone https://github.com/healtronlabs/gofasta.git
cd gofasta/packages/cli
go build -o gofasta
```

## Commands

### Project Management

#### `gofasta new [project-name]`

Create a new Gofasta project with enterprise architecture patterns.

```bash
# Create a basic API project
gofasta new my-api

# Create with specific database
gofasta new my-api --database mongodb

# Create from template
gofasta new my-service --template microservice

# Skip git initialization
gofasta new my-api --skip-git
```

**Available Templates:**
- `api` - REST API with CRUD operations
- `microservice` - Microservice with gRPC support
- `web` - Web API with static file serving
- `minimal` - Minimal setup with core modules
- `e-commerce` - E-commerce backend API template

**Database Options:**
- `postgresql` (default)
- `mongodb`
- `mysql`
- `sqlite`

### Code Generation

#### `gofasta generate [type] [name]`

Generate code components using Gofasta patterns.

```bash
# Generate a controller with CRUD operations
gofasta generate controller User --crud

# Generate a service
gofasta generate service User

# Generate a model with validation
gofasta generate model Product --validation

# Generate a repository
gofasta generate repository User --model User

# Generate middleware
gofasta generate middleware Auth

# Generate a complete module
gofasta generate module Payment
```

### Development Server

#### `gofasta dev`

Start development server with hot reload.

```bash
# Start with default settings
gofasta dev

# Custom port
gofasta dev --port 3000

# Watch specific directories
gofasta dev --watch ./internal --watch ./pkg

# Exclude paths from watching
gofasta dev --exclude tmp --exclude dist

# Custom build command
gofasta dev --build "go build -o tmp/app cmd/main.go"
```

**Features:**
- Hot reload on file changes
- Automatic rebuild and restart
- Environment variable loading
- Live reload for web assets

### Database Migrations

#### `gofasta migration [command]`

Manage database migrations.

```bash
# Create a new migration
gofasta migration create create_users_table

# Run pending migrations
gofasta migration run

# Run specific number of migrations
gofasta migration run --steps 2

# Rollback migrations
gofasta migration rollback

# Check migration status
gofasta migration status

# Reset all migrations (dangerous)
gofasta migration reset --force
```

### Build and Deployment

#### `gofasta build`

Build the application with optimizations.

```bash
# Build for current platform
gofasta build

# Cross-platform build
gofasta build --platform linux --arch amd64

# Custom output name
gofasta build --output my-app

# Build with custom ldflags
gofasta build --ldflags "-X main.Version=1.0.0"

# Build with tags
gofasta build --tags "production"

# Disable optimizations
gofasta build --optimize=false
```

### Testing

#### `gofasta test`

Run tests with Gofasta testing utilities.

```bash
# Run all tests
gofasta test

# Run with coverage
gofasta test --coverage

# Run specific packages
gofasta test --packages ./internal/services

# Run with race detection
gofasta test --race

# Run benchmarks
gofasta test --bench

# Verbose output
gofasta test --verbose

# JSON output
gofasta test --format json
```

## Configuration

### Environment Variables

The CLI respects these environment variables:

- `DATABASE_URL` - Default database connection string
- `GOFASTA_CONFIG` - Path to configuration file
- `GOFASTA_ENV` - Environment (development, production)

### Configuration File

Create a `.gofasta.yaml` file in your project root:

```yaml
database:
  url: "postgresql://localhost:5432/myapp"
  migrations_dir: "migrations"

dev:
  port: 8080
  watch:
    - "./internal"
    - "./pkg"
  exclude:
    - "tmp"
    - "dist"
    - ".git"

build:
  output: "app"
  optimize: true
  ldflags: "-X main.Version={{.Version}}"

test:
  coverage: true
  race: true
  verbose: false
```

## Examples

### Creating a New Project

```bash
# Create a new API project
gofasta new my-blog-api --database postgresql

cd my-blog-api

# Generate models
gofasta generate model Post --validation
gofasta generate model User --validation

# Generate services
gofasta generate service Post
gofasta generate service User

# Generate controllers with CRUD
gofasta generate controller Post --crud
gofasta generate controller User --crud

# Start development server
gofasta dev
```

### Setting Up Migrations

```bash
# Create initial migration
gofasta migration create create_users_table

# Edit the migration files in migrations/
# Then run migrations
gofasta migration run

# Create another migration
gofasta migration create create_posts_table
gofasta migration run
```

### Building for Production

```bash
# Build optimized binary
gofasta build --optimize

# Cross-compile for Linux
gofasta build --platform linux --arch amd64 --output app-linux

# Build with version info
gofasta build --ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

## Integration with IDEs

### VS Code

Install the Gofasta extension for enhanced development experience:

```json
{
  "gofasta.autoGenerate": true,
  "gofasta.liveReload": true,
  "gofasta.templatePath": "./templates"
}
```

### GoLand/IntelliJ

Configure external tools for Gofasta commands:

1. Go to Settings → Tools → External Tools
2. Add new tool with Gofasta CLI commands
3. Set up keyboard shortcuts for common operations

## Troubleshooting

### Common Issues

**Build Failures:**
```bash
# Clear module cache
go clean -modcache

# Verify go.mod
go mod verify && go mod tidy
```

**Development Server Issues:**
```bash
# Check port availability
lsof -i :8080

# Clear tmp directory
rm -rf tmp/
```

**Migration Errors:**
```bash
# Check database connection
gofasta migration status

# Reset and rerun
gofasta migration reset --force
gofasta migration run
```

### Debug Mode

Enable debug output:

```bash
export GOFASTA_DEBUG=true
gofasta dev --verbose
```

## Contributing

See the main Gofasta repository for contribution guidelines.

## License

MIT License - see the main Gofasta repository for details.