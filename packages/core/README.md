# @gofasta/core

The core module of the Gofasta framework providing dependency injection, module system, and application bootstrap functionality.

## Features

- **Dependency Injection Container**: Type-safe DI with lifecycle management
- **Module System**: Modular architecture with clear boundaries
- **Application Bootstrap**: Framework initialization and configuration
- **Provider Pattern**: Service registration and resolution
- **Lifecycle Management**: Singleton, transient, and scoped services

## Installation

```bash
go get github.com/healtronlabs/gofasta/packages/core
```

## Quick Start

```go
package main

import (
    "github.com/healtronlabs/gofasta/packages/core"
)

@Module{
    Providers: []interface{}{&MyService{}},
}
type AppModule struct{}

func main() {
    app := core.CreateApp(&AppModule{})
    app.Start()
}
```

## Documentation

See the [full documentation](../../docs/architecture/dependency-injection.md) for detailed usage instructions.