package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"
)

// Application is the main application interface for Gofasta
type Application interface {
	RegisterModule(module Module) error
	Start() error
	Stop() error
	GetService(serviceType reflect.Type) (interface{}, error)
	GetServiceByName(name string) (interface{}, error)
	Listen(port int) error
	UseGlobalPipes(pipes ...Pipe) error
	UseGlobalGuards(guards ...Guard) error
	UseGlobalInterceptors(interceptors ...Interceptor) error
	UseGlobalFilters(filters ...ExceptionFilter) error
	GetConfig() *ApplicationConfig
	GetContext() context.Context
	CreateScope(scopeId string) *ScopedContext
	DestroyScope(scopeId string) error
	Shutdown(timeout time.Duration) error
}

// ApplicationConfig holds application configuration
type ApplicationConfig struct {
	Port            int                    `json:"port" yaml:"port"`
	Host            string                 `json:"host" yaml:"host"`
	Environment     string                 `json:"environment" yaml:"environment"`
	LogLevel        string                 `json:"logLevel" yaml:"logLevel"`
	EnableCORS      bool                   `json:"enableCORS" yaml:"enableCORS"`
	EnableMetrics   bool                   `json:"enableMetrics" yaml:"enableMetrics"`
	EnableTracing   bool                   `json:"enableTracing" yaml:"enableTracing"`
	ShutdownTimeout time.Duration          `json:"shutdownTimeout" yaml:"shutdownTimeout"`
	Custom          map[string]interface{} `json:"custom" yaml:"custom"`
}

// DefaultApplicationConfig returns default application configuration
func DefaultApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{
		Port:            8080,
		Host:            "localhost",
		Environment:     "development",
		LogLevel:        "info",
		EnableCORS:      true,
		EnableMetrics:   false,
		EnableTracing:   false,
		ShutdownTimeout: 30 * time.Second,
		Custom:          make(map[string]interface{}),
	}
}

// GofastaApplication is the default implementation of Application
type GofastaApplication struct {
	container          *DIContainer
	modules            []Module
	config             *ApplicationConfig
	globalPipes        []Pipe
	globalGuards       []Guard
	globalInterceptors []Interceptor
	globalFilters      []ExceptionFilter
	isStarted          bool
	isShuttingDown     bool
	shutdownFn         func() error
	ctx                context.Context
	cancel             context.CancelFunc
	mutex              sync.RWMutex
	startTime          time.Time
}

// CreateApp creates a new Gofasta application with the specified root module
func CreateApp(rootModule Module, config ...*ApplicationConfig) Application {
	cfg := DefaultApplicationConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	// Override with environment variables
	if port := os.Getenv("PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}
	if host := os.Getenv("HOST"); host != "" {
		cfg.Host = host
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	container := NewDIContainer()
	app := &GofastaApplication{
		container:          container,
		modules:            make([]Module, 0),
		config:             cfg,
		globalPipes:        make([]Pipe, 0),
		globalGuards:       make([]Guard, 0),
		globalInterceptors: make([]Interceptor, 0),
		globalFilters:      make([]ExceptionFilter, 0),
		ctx:                ctx,
		cancel:             cancel,
	}

	// Register the application itself as a service
	container.RegisterInstance(reflect.TypeOf((*Application)(nil)).Elem(), app)
	container.RegisterInstance(reflect.TypeOf((*ApplicationConfig)(nil)).Elem(), cfg)

	// Register the root module
	if err := app.RegisterModule(rootModule); err != nil {
		panic(fmt.Sprintf("Failed to register root module: %v", err))
	}

	return app
}

// RegisterModule registers a module with the application
func (app *GofastaApplication) RegisterModule(module Module) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if app.isStarted {
		return fmt.Errorf("cannot register modules after application has started")
	}

	app.modules = append(app.modules, module)

	// Configure the module with the DI container
	if err := module.Configure(app.container); err != nil {
		return fmt.Errorf("failed to configure module: %w", err)
	}

	return nil
}

// Start initializes and starts the application
func (app *GofastaApplication) Start() error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if app.isStarted {
		return fmt.Errorf("application is already started")
	}

	app.startTime = time.Now()

	// Initialize all dependencies
	if err := app.container.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	// Initialize all modules
	for _, module := range app.modules {
		if err := module.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize module: %w", err)
		}
	}

	app.isStarted = true

	// Log startup information
	fmt.Printf("🚀 Gofasta application started successfully\n")
	fmt.Printf("   Environment: %s\n", app.config.Environment)
	fmt.Printf("   Log Level: %s\n", app.config.LogLevel)
	fmt.Printf("   Startup Time: %v\n", time.Since(app.startTime))
	fmt.Printf("   Modules: %d\n", len(app.modules))

	return nil
}

// Stop gracefully stops the application
func (app *GofastaApplication) Stop() error {
	return app.Shutdown(app.config.ShutdownTimeout)
}

// Shutdown gracefully shuts down the application with timeout
func (app *GofastaApplication) Shutdown(timeout time.Duration) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if !app.isStarted || app.isShuttingDown {
		return nil
	}

	app.isShuttingDown = true
	fmt.Println("🛑 Shutting down Gofasta application...")

	// Cancel application context
	app.cancel()

	// HTTP server shutdown should be handled by HTTP module

	// Execute custom shutdown function
	if app.shutdownFn != nil {
		if err := app.shutdownFn(); err != nil {
			fmt.Printf("Error executing custom shutdown function: %v\n", err)
		}
	}

	// Cleanup modules in reverse order
	for i := len(app.modules) - 1; i >= 0; i-- {
		if err := app.modules[i].Cleanup(); err != nil {
			fmt.Printf("Error cleaning up module: %v\n", err)
		}
	}

	// Shutdown DI container
	if err := app.container.Shutdown(timeout); err != nil {
		fmt.Printf("Error shutting down DI container: %v\n", err)
	}

	app.isStarted = false
	app.isShuttingDown = false

	fmt.Printf("✅ Gofasta application shut down successfully (uptime: %v)\n", time.Since(app.startTime))
	return nil
}

// GetService retrieves a service from the DI container
func (app *GofastaApplication) GetService(serviceType reflect.Type) (interface{}, error) {
	return app.container.ResolveWithContext(app.ctx, serviceType)
}

// GetServiceByName retrieves a named service from the DI container
func (app *GofastaApplication) GetServiceByName(name string) (interface{}, error) {
	return app.container.ResolveNamedWithContext(app.ctx, name)
}

// UseGlobalPipes adds global pipes that will be applied to all requests
func (app *GofastaApplication) UseGlobalPipes(pipes ...Pipe) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	app.globalPipes = append(app.globalPipes, pipes...)
	return nil
}

// UseGlobalGuards adds global guards that will be applied to all requests
func (app *GofastaApplication) UseGlobalGuards(guards ...Guard) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	app.globalGuards = append(app.globalGuards, guards...)
	return nil
}

// UseGlobalInterceptors adds global interceptors that will be applied to all requests
func (app *GofastaApplication) UseGlobalInterceptors(interceptors ...Interceptor) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	app.globalInterceptors = append(app.globalInterceptors, interceptors...)
	return nil
}

// UseGlobalFilters adds global exception filters that will be applied to all requests
func (app *GofastaApplication) UseGlobalFilters(filters ...ExceptionFilter) error {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	app.globalFilters = append(app.globalFilters, filters...)
	return nil
}

// GetConfig returns the application configuration
func (app *GofastaApplication) GetConfig() *ApplicationConfig {
	return app.config
}

// GetContext returns the application context
func (app *GofastaApplication) GetContext() context.Context {
	return app.ctx
}

// CreateScope creates a new scoped context
func (app *GofastaApplication) CreateScope(scopeId string) *ScopedContext {
	return app.container.CreateScope(scopeId)
}

// DestroyScope destroys a scoped context
func (app *GofastaApplication) DestroyScope(scopeId string) error {
	return app.container.DestroyScope(scopeId)
}

// Listen delegates to HTTP module for server functionality
func (app *GofastaApplication) Listen(port int) error {
	if !app.isStarted {
		if err := app.Start(); err != nil {
			return err
		}
	}

	// Update port in config
	app.config.Port = port

	// HTTP server functionality should be handled by HTTP module
	// This is a placeholder - actual implementation should use HTTP module
	fmt.Printf("🌐 Gofasta application ready to listen on port %d\n", port)
	fmt.Println("Note: HTTP server functionality should be implemented via HTTP module")
	
	// Block to prevent application from exiting
	select {}
}

// setupGracefulShutdown sets up graceful shutdown handling
func (app *GofastaApplication) setupGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n🔄 Received shutdown signal...")

	if err := app.Shutdown(app.config.ShutdownTimeout); err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
		os.Exit(1)
	}
}

// SetShutdownHook sets a custom shutdown hook
func (app *GofastaApplication) SetShutdownHook(fn func() error) {
	app.mutex.Lock()
	defer app.mutex.Unlock()
	app.shutdownFn = fn
}

// GetGlobalPipes returns the global pipes
func (app *GofastaApplication) GetGlobalPipes() []Pipe {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.globalPipes
}

// GetGlobalGuards returns the global guards
func (app *GofastaApplication) GetGlobalGuards() []Guard {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.globalGuards
}

// GetGlobalInterceptors returns the global interceptors
func (app *GofastaApplication) GetGlobalInterceptors() []Interceptor {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.globalInterceptors
}

// GetGlobalFilters returns the global exception filters
func (app *GofastaApplication) GetGlobalFilters() []ExceptionFilter {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.globalFilters
}

// IsStarted returns whether the application is started
func (app *GofastaApplication) IsStarted() bool {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.isStarted
}

// IsShuttingDown returns whether the application is shutting down
func (app *GofastaApplication) IsShuttingDown() bool {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.isShuttingDown
}

// GetUptime returns the application uptime
func (app *GofastaApplication) GetUptime() time.Duration {
	if !app.isStarted {
		return 0
	}
	return time.Since(app.startTime)
}

// GetModules returns the registered modules
func (app *GofastaApplication) GetModules() []Module {
	app.mutex.RLock()
	defer app.mutex.RUnlock()
	return app.modules
}