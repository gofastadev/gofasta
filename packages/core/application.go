package core

import (
	"fmt"
	"reflect"
)

// Application is the main application interface for Gofasta
type Application interface {
	RegisterModule(module Module) error
	Start() error
	Stop() error
	GetService(serviceType reflect.Type) (interface{}, error)
	Listen(port int) error
	UseGlobalPipes(pipes ...Pipe) error
	UseGlobalGuards(guards ...Guard) error
	UseGlobalInterceptors(interceptors ...Interceptor) error
}

// GofastaApplication is the default implementation of Application
type GofastaApplication struct {
	container          *DIContainer
	modules            []Module
	globalPipes        []Pipe
	globalGuards       []Guard
	globalInterceptors []Interceptor
	isStarted          bool
	shutdownFn         func() error
}

// CreateApp creates a new Gofasta application with the specified root module
func CreateApp(rootModule Module) Application {
	container := NewDIContainer()
	app := &GofastaApplication{
		container: container,
		modules:   make([]Module, 0),
	}

	// Register the root module
	app.RegisterModule(rootModule)

	return app
}

// RegisterModule registers a module with the application
func (app *GofastaApplication) RegisterModule(module Module) error {
	if app.isStarted {
		return fmt.Errorf("cannot register modules after application has started")
	}

	app.modules = append(app.modules, module)

	// Configure the module with the DI container
	if err := module.Configure(app.container); err != nil {
		return fmt.Errorf("failed to configure module: %w", err)
	}

	// Register providers
	for _, provider := range module.GetProviders() {
		if err := app.container.RegisterProvider(provider); err != nil {
			return fmt.Errorf("failed to register provider: %w", err)
		}
	}

	// Register controllers
	for _, controller := range module.GetControllers() {
		if err := app.container.RegisterController(controller); err != nil {
			return fmt.Errorf("failed to register controller: %w", err)
		}
	}

	// Register imported modules
	for _, importedModule := range module.GetImports() {
		if err := app.RegisterModule(importedModule); err != nil {
			return fmt.Errorf("failed to register imported module: %w", err)
		}
	}

	return nil
}

// Start initializes and starts the application
func (app *GofastaApplication) Start() error {
	if app.isStarted {
		return fmt.Errorf("application is already started")
	}

	// Initialize all dependencies
	if err := app.container.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	app.isStarted = true
	return nil
}

// Stop gracefully stops the application
func (app *GofastaApplication) Stop() error {
	if !app.isStarted {
		return nil
	}

	var err error
	if app.shutdownFn != nil {
		err = app.shutdownFn()
	}

	app.isStarted = false
	return err
}

// GetService retrieves a service from the DI container
func (app *GofastaApplication) GetService(serviceType reflect.Type) (interface{}, error) {
	return app.container.Resolve(serviceType)
}

// UseGlobalPipes adds global pipes that will be applied to all requests
func (app *GofastaApplication) UseGlobalPipes(pipes ...Pipe) error {
	app.globalPipes = append(app.globalPipes, pipes...)
	return nil
}

// UseGlobalGuards adds global guards that will be applied to all requests
func (app *GofastaApplication) UseGlobalGuards(guards ...Guard) error {
	app.globalGuards = append(app.globalGuards, guards...)
	return nil
}

// UseGlobalInterceptors adds global interceptors that will be applied to all requests
func (app *GofastaApplication) UseGlobalInterceptors(interceptors ...Interceptor) error {
	app.globalInterceptors = append(app.globalInterceptors, interceptors...)
	return nil
}

// Listen starts the HTTP server on the specified port
func (app *GofastaApplication) Listen(port int) error {
	if !app.isStarted {
		if err := app.Start(); err != nil {
			return err
		}
	}

	// This will be implemented in the HTTP module
	// For now, just a placeholder
	fmt.Printf("Gofasta application listening on port %d\n", port)

	// Block forever (in real implementation, this would start the HTTP server)
	select {}
}
