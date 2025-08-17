# Upgrading Guide

This guide covers upgrading Gofasta framework and CLI tools to newer versions.

## Overview

Gofasta follows semantic versioning (SemVer) with clear upgrade paths:

- **Patch updates** (1.0.x → 1.0.y) - Bug fixes, always safe
- **Minor updates** (1.x.0 → 1.y.0) - New features, backward compatible
- **Major updates** (1.x.x → 2.0.0) - Breaking changes, requires migration

## Before Upgrading

### 1. Backup Your Project

```bash
# Create a backup branch
git checkout -b backup-before-upgrade
git push origin backup-before-upgrade

# Or create a full backup
cp -r your-project your-project-backup
```

### 2. Check Current Versions

```bash
# Check CLI version
gofasta version

# Check framework versions in your project
go list -m all | grep gofasta

# Check for outdated dependencies
go list -u -m all | grep gofasta
```

### 3. Review Changelog

Before upgrading, review the [CHANGELOG.md](../CHANGELOG.md) to understand:
- New features
- Breaking changes
- Deprecation notices
- Migration requirements

## Upgrading CLI Tool

### To Latest Version

```bash
# Upgrade to latest version
go install github.com/healtronlabs/gofasta/packages/cli@latest

# Verify upgrade
gofasta version
```

### To Specific Version

```bash
# Upgrade to specific version
go install github.com/healtronlabs/gofasta/packages/cli@v1.2.0

# Verify version
gofasta version
```

### From Pre-built Binaries

```bash
# Download latest release
curl -L https://github.com/healtronlabs/gofasta/releases/latest/download/gofasta-linux-amd64 -o gofasta-new

# Replace existing binary
sudo mv gofasta-new /usr/local/bin/gofasta
sudo chmod +x /usr/local/bin/gofasta

# Verify upgrade
gofasta version
```

## Upgrading Framework in Projects

### Automatic Upgrade (Recommended)

Use Gofasta CLI to upgrade your project:

```bash
# Navigate to your project
cd your-gofasta-project

# Check for available updates
gofasta upgrade check

# Upgrade to latest compatible version
gofasta upgrade

# Or upgrade to specific version
gofasta upgrade --version v1.2.0
```

### Manual Upgrade

#### Step 1: Update Dependencies

```bash
# Update all Gofasta packages
go get -u github.com/healtronlabs/gofasta/packages/core@latest
go get -u github.com/healtronlabs/gofasta/packages/http@latest
go get -u github.com/healtronlabs/gofasta/packages/orm@latest
go get -u github.com/healtronlabs/gofasta/packages/auth@latest
go get -u github.com/healtronlabs/gofasta/packages/validation@latest

# Clean up dependencies
go mod tidy
```

#### Step 2: Update Imports (if needed)

Some major versions may require import path changes:

```go
// Old import (v1.x)
import "github.com/healtronlabs/gofasta/packages/core"

// New import (v2.x) - example
import "github.com/healtronlabs/gofasta/v2/packages/core"
```

#### Step 3: Run Tests

```bash
# Run tests to check for breaking changes
go test ./...

# Run with Gofasta test utilities
gofasta test --coverage
```

#### Step 4: Update Configuration

Some versions may require configuration updates:

```go
// Check for deprecated configuration options
// Update according to migration guide
```

## Version-Specific Upgrade Guides

### Upgrading from v0.9.x to v1.0.x

#### Breaking Changes

1. **Module Structure Changes**
   ```go
   // Old (v0.9.x)
   import "github.com/healtronlabs/gofasta/core"
   
   // New (v1.0.x)
   import "github.com/healtronlabs/gofasta/packages/core"
   ```

2. **Configuration API Changes**
   ```go
   // Old
   app := core.NewApplication()
   
   // New
   app := core.CreateApp(&AppModule{})
   ```

3. **Database Connection Changes**
   ```go
   // Old
   orm.Connect("postgresql://...")
   
   // New
   app.RegisterModule(orm.NewGofastaOrmModuleFromURL("postgresql://..."))
   ```

#### Migration Steps

1. **Update imports**:
   ```bash
   # Use find and replace or IDE refactoring
   find . -name "*.go" -exec sed -i 's|github.com/healtronlabs/gofasta/core|github.com/healtronlabs/gofasta/packages/core|g' {} +
   ```

2. **Update application initialization**:
   ```go
   // Create AppModule
   type AppModule struct {
       *core.BaseModule
   }
   
   func (m *AppModule) Configure() {
       // Move your configuration here
   }
   
   // Update main function
   func main() {
       app := core.CreateApp(&AppModule{})
       // Register modules...
   }
   ```

3. **Update database configuration**:
   ```go
   // Replace database initialization code
   app.RegisterModule(orm.NewGofastaOrmModuleFromURL(databaseURL))
   ```

### Upgrading from v1.0.x to v1.1.x

This is a minor version upgrade with new features but no breaking changes.

#### New Features
- Enhanced validation rules
- New middleware options
- Improved error handling

#### Migration Steps
```bash
# Simple dependency update
go get -u github.com/healtronlabs/gofasta/packages/...@v1.1.0
go mod tidy
```

## Automated Migration Tools

### Gofasta Migration Assistant

```bash
# Install migration tool
go install github.com/healtronlabs/gofasta/tools/migrate@latest

# Run migration analysis
gofasta-migrate analyze

# Apply automatic migrations
gofasta-migrate apply --from v0.9.0 --to v1.0.0

# Review changes
git diff
```

### IDE Refactoring Tools

#### VS Code

1. Install Gofasta extension
2. Use Command Palette: "Gofasta: Migrate Project"
3. Select target version
4. Review and apply changes

#### GoLand

1. Use "Refactor → Migrate Gofasta Project"
2. Select source and target versions
3. Apply suggested changes

## Testing After Upgrade

### 1. Run Comprehensive Tests

```bash
# Run all tests
gofasta test --coverage --race

# Run specific test suites
gofasta test --packages ./internal/services
gofasta test --packages ./internal/controllers
```

### 2. Integration Tests

```bash
# Start test database
docker-compose up -d test-db

# Run integration tests
gofasta test --tags integration

# Run end-to-end tests
gofasta test --tags e2e
```

### 3. Performance Tests

```bash
# Run benchmarks
gofasta test --bench

# Load testing (if available)
gofasta test --load
```

### 4. Manual Testing

```bash
# Start development server
gofasta dev

# Test key endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/users
```

## Rollback Procedures

If upgrade causes issues, you can rollback:

### Git Rollback

```bash
# Rollback to previous version
git checkout backup-before-upgrade

# Or reset to specific commit
git reset --hard <commit-hash>

# Force push if needed (be careful)
git push --force-with-lease
```

### Dependency Rollback

```bash
# Rollback to previous versions
go get github.com/healtronlabs/gofasta/packages/core@v1.0.0
go get github.com/healtronlabs/gofasta/packages/http@v1.0.0
go mod tidy
```

### CLI Rollback

```bash
# Install previous CLI version
go install github.com/healtronlabs/gofasta/packages/cli@v1.0.0
```

## Upgrade Checklist

Before upgrading:
- [ ] Backup project and database
- [ ] Review changelog and breaking changes
- [ ] Check compatibility matrix
- [ ] Plan downtime if needed

During upgrade:
- [ ] Update CLI tool first
- [ ] Update framework dependencies
- [ ] Apply code migrations
- [ ] Update configuration files
- [ ] Run comprehensive tests

After upgrade:
- [ ] Verify all functionality works
- [ ] Monitor application performance
- [ ] Update documentation
- [ ] Train team on new features
- [ ] Deploy to staging environment first

## Getting Help During Upgrades

### Resources

1. **Migration Guides**: Check version-specific guides in `/docs/migrations/`
2. **Community Forum**: Ask questions at [community.gofasta.dev](https://community.gofasta.dev)
3. **GitHub Issues**: Report problems at [github.com/healtronlabs/gofasta/issues](https://github.com/healtronlabs/gofasta/issues)
4. **Discord Support**: Real-time help at [discord.gg/gofasta](https://discord.gg/gofasta)

### Professional Support

For enterprise customers:
- **Migration Assistance**: Guided upgrade support
- **Custom Migration Tools**: Tailored automation
- **Priority Support**: Direct access to core team
- **Training Sessions**: Team training on new features

Contact: enterprise@healtronlabs.com

## Best Practices

### 1. Regular Updates

```bash
# Check for updates weekly
gofasta upgrade check

# Apply patch updates immediately
gofasta upgrade --patch-only

# Plan minor/major updates quarterly
```

### 2. Staged Rollouts

1. **Development Environment** - Test new version
2. **Staging Environment** - Validate with production-like data  
3. **Canary Deployment** - Limited production rollout
4. **Full Production** - Complete rollout

### 3. Monitoring

After upgrades, monitor:
- Application performance metrics
- Error rates and logs
- Database performance
- Memory and CPU usage

### 4. Documentation

Keep upgrade history:
```markdown
## Upgrade History

- 2024-01-15: v0.9.2 → v1.0.0 (Major upgrade, 2 hours downtime)
- 2024-02-01: v1.0.0 → v1.0.1 (Patch, no downtime)
- 2024-03-01: v1.0.1 → v1.1.0 (Minor, 30min validation)
```

## Staying Updated

### Release Notifications

- **GitHub**: Watch the repository for releases
- **Newsletter**: Subscribe to Gofasta updates
- **RSS Feed**: Follow the release feed
- **Discord**: Join announcement channel

### Compatibility Planning

Plan upgrades around:
- Go version releases
- Dependency updates
- Security patches
- Feature requirements

For more information, see the [Release Schedule](release-schedule.md).