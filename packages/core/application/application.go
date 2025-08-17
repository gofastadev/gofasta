package application

import (
	"context"
	"reflect"

	"github.com/healtronlabs/gofasta/packages/core/container"
	"github.com/healtronlabs/gofasta/packages/core/module"
)

// Application represents the main Gofasta application
type Application struct {
	container *container.DIContainer
	modules   []module.Module
	config    *Config
}

// Config holds application configuration
type Config struct {
	Port        int
	Environment string
	LogLevel    string
}

// CreateApp creates a new Gofasta application
func CreateApp(rootModule module.Module, config ...*Config) *Application {
	cfg := &Config{
		Port:        8080,
		Environment: "development",
		LogLevel:    "info",
	}
	
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	app := &Application{
		container: container.NewDIContainer(),
		config:    cfg,
	}

	app.RegisterModule(rootModule)
	return app
}

// RegisterModule registers a module with the application
func (app *Application) RegisterModule(mod module.Module) error {
	app.modules = append(app.modules, mod)
	return mod.Configure(app.container)
}

// GetService retrieves a service from the DI container
func (app *Application) GetService(serviceType reflect.Type) (interface{}, error) {
	return app.container.Resolve(serviceType)
}

// Start starts the application
func (app *Application) Start() error {
	// Initialize all modules
	for _, mod := range app.modules {
		if err := mod.Initialize(); err != nil {
			return err
		}
	}
	
	return nil
}

// Stop gracefully stops the application
func (app *Application) Stop(ctx context.Context) error {
	// Cleanup all modules
	for _, mod := range app.modules {
		if err := mod.Cleanup(); err != nil {
			return err
		}
	}
	
	return nil
}

// Listen starts the HTTP server (will be implemented in http package)
func (app *Application) Listen(port int) error {
	// This will be implemented when we integrate with the HTTP package
	return nil
}