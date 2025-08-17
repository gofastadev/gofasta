# @gofasta/http

HTTP module for the Gofasta framework providing web server capabilities, routing, middleware, and WebSocket support.

## Features

- **High-Performance HTTP Server**: Built on Go's net/http
- **Flexible Routing**: Struct tag-based routing with parameter extraction
- **Middleware Pipeline**: Guards, interceptors, and custom middleware
- **WebSocket Support**: Real-time communication capabilities
- **Request/Response Processing**: Automatic serialization and validation
- **Static File Serving**: Built-in static file server with compression

## Installation

```bash
go get github.com/healtronlabs/gofasta/packages/http
```

## Quick Start

```go
package main

import (
    "github.com/healtronlabs/gofasta/packages/core"
    "github.com/healtronlabs/gofasta/packages/http"
)

@Controller{Path: "/api/users"}
type UserController struct {
    UserService *UserService `inject:""`
}

@Get{Path: "/:id"}
func (c *UserController) GetUser(@Param("id") id string) (*User, error) {
    return c.UserService.FindById(id)
}

@Module{
    Controllers: []interface{}{&UserController{}},
    Imports:     []interface{}{&http.HttpModule{}},
}
type AppModule struct{}

func main() {
    app := core.CreateApp(&AppModule{})
    app.Listen(8080)
}
```