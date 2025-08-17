package core

import (
	"os"
	"reflect"
	"testing"
	"time"
)

// Test module for application testing
type TestAppModule struct {
	BaseModule
}

func (m *TestAppModule) Configure(container *DIContainer) error {
	// Register test services
	container.RegisterProvider(&TestLogger{})
	container.RegisterProvider(&TestUserService{})
	
	config := &TestConfig{
		DatabaseURL: "test://localhost",
		APIKey:      "test-key",
	}
	container.RegisterInstance(reflect.TypeOf((*TestConfig)(nil)).Elem(), config)
	container.namedServices["test-config"] = container.services[reflect.TypeOf((*TestConfig)(nil)).Elem()]
	
	return nil
}

func TestDefaultApplicationConfig(t *testing.T) {
	config := DefaultApplicationConfig()
	
	if config == nil {
		t.Fatal("DefaultApplicationConfig() returned nil")
	}
	
	if config.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Port)
	}
	
	if config.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got %s", config.Host)
	}
	
	if config.Environment != "development" {
		t.Errorf("Expected default environment 'development', got %s", config.Environment)
	}
	
	if config.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", config.LogLevel)
	}
	
	if !config.EnableCORS {
		t.Error("Expected CORS to be enabled by default")
	}
	
	if config.EnableMetrics {
		t.Error("Expected metrics to be disabled by default")
	}
	
	if config.ShutdownTimeout != 30*time.Second {
		t.Errorf("Expected default shutdown timeout 30s, got %v", config.ShutdownTimeout)
	}
	
	if config.Custom == nil {
		t.Error("Expected custom map to be initialized")
	}
}

func TestCreateApp(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	if app == nil {
		t.Fatal("CreateApp() returned nil")
	}
	
	gofastaApp, ok := app.(*GofastaApplication)
	if !ok {
		t.Fatal("CreateApp() did not return *GofastaApplication")
	}
	
	if gofastaApp.container == nil {
		t.Error("Application container not initialized")
	}
	
	if len(gofastaApp.modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(gofastaApp.modules))
	}
	
	if gofastaApp.config == nil {
		t.Error("Application config not initialized")
	}
	
	if gofastaApp.isStarted {
		t.Error("Application should not be started by default")
	}
}

func TestCreateAppWithConfig(t *testing.T) {
	module := &TestAppModule{}
	config := &ApplicationConfig{
		Port:        9000,
		Host:        "0.0.0.0",
		Environment: "test",
		LogLevel:    "debug",
	}
	
	app := CreateApp(module, config)
	
	if app == nil {
		t.Fatal("CreateApp() returned nil")
	}
	
	appConfig := app.GetConfig()
	if appConfig.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", appConfig.Port)
	}
	
	if appConfig.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got %s", appConfig.Host)
	}
	
	if appConfig.Environment != "test" {
		t.Errorf("Expected environment 'test', got %s", appConfig.Environment)
	}
}

func TestCreateAppWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("PORT", "3000")
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("LOG_LEVEL", "error")
	
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("HOST")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("LOG_LEVEL")
	}()
	
	module := &TestAppModule{}
	app := CreateApp(module)
	
	config := app.GetConfig()
	if config.Port != 3000 {
		t.Errorf("Expected port from env var 3000, got %d", config.Port)
	}
	
	if config.Host != "127.0.0.1" {
		t.Errorf("Expected host from env var '127.0.0.1', got %s", config.Host)
	}
	
	if config.Environment != "production" {
		t.Errorf("Expected environment from env var 'production', got %s", config.Environment)
	}
	
	if config.LogLevel != "error" {
		t.Errorf("Expected log level from env var 'error', got %s", config.LogLevel)
	}
}

func TestApplication_RegisterModule(t *testing.T) {
	module1 := &TestAppModule{}
	app := CreateApp(module1)
	
	gofastaApp := app.(*GofastaApplication)
	
	// Test registering additional module
	module2 := &TestAppModule{}
	err := app.RegisterModule(module2)
	if err != nil {
		t.Fatalf("Failed to register module: %v", err)
	}
	
	if len(gofastaApp.modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(gofastaApp.modules))
	}
	
	// Test registering module after start should fail
	app.Start()
	defer app.Stop()
	
	module3 := &TestAppModule{}
	err = app.RegisterModule(module3)
	if err == nil {
		t.Error("Expected error when registering module after start")
	}
}

func TestApplication_Start(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Test starting application
	err := app.Start()
	if err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	
	gofastaApp := app.(*GofastaApplication)
	if !gofastaApp.isStarted {
		t.Error("Application not marked as started")
	}
	
	// Test double start should not fail
	err = app.Start()
	if err == nil {
		t.Error("Expected error on double start")
	}
	
	// Cleanup
	app.Stop()
}

func TestApplication_Stop(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Start and then stop
	err := app.Start()
	if err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	
	err = app.Stop()
	if err != nil {
		t.Fatalf("Failed to stop application: %v", err)
	}
	
	gofastaApp := app.(*GofastaApplication)
	if gofastaApp.isStarted {
		t.Error("Application still marked as started after stop")
	}
}

func TestApplication_Shutdown(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Start application
	err := app.Start()
	if err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	
	// Test shutdown with timeout
	err = app.Shutdown(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to shutdown application: %v", err)
	}
	
	gofastaApp := app.(*GofastaApplication)
	if gofastaApp.isStarted {
		t.Error("Application still marked as started after shutdown")
	}
}

func TestApplication_GetService(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	err := app.Start()
	if err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer app.Stop()
	
	// Test getting service
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	instance, err := app.GetService(loggerType)
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}
	
	logger, ok := instance.(*TestLogger)
	if !ok {
		t.Error("Service is not of correct type")
	}
	
	if logger.Level != "INFO" {
		t.Error("Service not properly initialized")
	}
}

func TestApplication_GetServiceByName(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	err := app.Start()
	if err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer app.Stop()
	
	// Test getting named service
	instance, err := app.GetServiceByName("test-config")
	if err != nil {
		t.Fatalf("Failed to get named service: %v", err)
	}
	
	config, ok := instance.(*TestConfig)
	if !ok {
		t.Error("Named service is not of correct type")
	}
	
	if config.DatabaseURL != "test://localhost" {
		t.Error("Named service not resolved correctly")
	}
}

func TestApplication_UseGlobalPipes(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Create test pipes
	pipe1 := &TestPipe{Name: "pipe1"}
	pipe2 := &TestPipe{Name: "pipe2"}
	
	err := app.UseGlobalPipes(pipe1, pipe2)
	if err != nil {
		t.Fatalf("Failed to add global pipes: %v", err)
	}
	
	gofastaApp := app.(*GofastaApplication)
	if len(gofastaApp.globalPipes) != 2 {
		t.Errorf("Expected 2 global pipes, got %d", len(gofastaApp.globalPipes))
	}
}

func TestApplication_UseGlobalGuards(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Create test guards
	guard1 := &TestGuard{Name: "guard1"}
	guard2 := &TestGuard{Name: "guard2"}
	
	err := app.UseGlobalGuards(guard1, guard2)
	if err != nil {
		t.Fatalf("Failed to add global guards: %v", err)
	}
	
	gofastaApp := app.(*GofastaApplication)
	if len(gofastaApp.globalGuards) != 2 {
		t.Errorf("Expected 2 global guards, got %d", len(gofastaApp.globalGuards))
	}
}

func TestApplication_UseGlobalInterceptors(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Create test interceptors
	interceptor1 := &TestInterceptor{Name: "interceptor1"}
	interceptor2 := &TestInterceptor{Name: "interceptor2"}
	
	err := app.UseGlobalInterceptors(interceptor1, interceptor2)
	if err != nil {
		t.Fatalf("Failed to add global interceptors: %v", err)
	}
	
	gofastaApp := app.(*GofastaApplication)
	if len(gofastaApp.globalInterceptors) != 2 {
		t.Errorf("Expected 2 global interceptors, got %d", len(gofastaApp.globalInterceptors))
	}
}

func TestApplication_UseGlobalFilters(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	// Create test filters
	filter1 := &TestExceptionFilter{Name: "filter1"}
	filter2 := &TestExceptionFilter{Name: "filter2"}
	
	err := app.UseGlobalFilters(filter1, filter2)
	if err != nil {
		t.Fatalf("Failed to add global filters: %v", err)
	}
	
	gofastaApp := app.(*GofastaApplication)
	if len(gofastaApp.globalFilters) != 2 {
		t.Errorf("Expected 2 global filters, got %d", len(gofastaApp.globalFilters))
	}
}

func TestApplication_CreateAndDestroyScope(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	err := app.Start()
	if err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	defer app.Stop()
	
	// Test creating scope
	scope := app.CreateScope("test-scope")
	if scope == nil {
		t.Error("Failed to create scope")
	}
	
	// Test destroying scope
	err = app.DestroyScope("test-scope")
	if err != nil {
		t.Errorf("Failed to destroy scope: %v", err)
	}
}

func TestApplication_GetConfig(t *testing.T) {
	module := &TestAppModule{}
	config := &ApplicationConfig{
		Port:        9000,
		Environment: "test",
	}
	app := CreateApp(module, config)
	
	retrievedConfig := app.GetConfig()
	if retrievedConfig != config {
		t.Error("GetConfig() did not return the same config instance")
	}
	
	if retrievedConfig.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", retrievedConfig.Port)
	}
}

func TestApplication_GetContext(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	
	ctx := app.GetContext()
	if ctx == nil {
		t.Error("GetContext() returned nil")
	}
	
	// Test that context is cancelled when application is stopped
	err := app.Start()
	if err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled while application is running")
	default:
		// Context is not cancelled, which is expected
	}
	
	app.Stop()
	
	// After stop, context should be cancelled
	select {
	case <-ctx.Done():
		// Context is cancelled, which is expected
	default:
		t.Error("Context should be cancelled after application stop")
	}
}

func TestGofastaApplication_AdditionalMethods(t *testing.T) {
	module := &TestAppModule{}
	app := CreateApp(module)
	gofastaApp := app.(*GofastaApplication)
	
	// Test IsStarted
	if gofastaApp.IsStarted() {
		t.Error("Application should not be started initially")
	}
	
	app.Start()
	if !gofastaApp.IsStarted() {
		t.Error("Application should be started after Start()")
	}
	
	// Test IsShuttingDown
	if gofastaApp.IsShuttingDown() {
		t.Error("Application should not be shutting down initially")
	}
	
	// Test GetUptime
	uptime := gofastaApp.GetUptime()
	if uptime <= 0 {
		t.Error("Uptime should be positive when application is started")
	}
	
	// Test GetModules
	modules := gofastaApp.GetModules()
	if len(modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(modules))
	}
	
	// Test GetGlobalPipes, GetGlobalGuards, etc.
	pipes := gofastaApp.GetGlobalPipes()
	if len(pipes) != 0 {
		t.Errorf("Expected 0 global pipes initially, got %d", len(pipes))
	}
	
	guards := gofastaApp.GetGlobalGuards()
	if len(guards) != 0 {
		t.Errorf("Expected 0 global guards initially, got %d", len(guards))
	}
	
	interceptors := gofastaApp.GetGlobalInterceptors()
	if len(interceptors) != 0 {
		t.Errorf("Expected 0 global interceptors initially, got %d", len(interceptors))
	}
	
	filters := gofastaApp.GetGlobalFilters()
	if len(filters) != 0 {
		t.Errorf("Expected 0 global filters initially, got %d", len(filters))
	}
	
	app.Stop()
	
	if gofastaApp.IsStarted() {
		t.Error("Application should not be started after Stop()")
	}
}

// Test helper types
type TestPipe struct {
	Name string
}

func (p *TestPipe) Transform(value interface{}, metadata *PipeMetadata) (interface{}, error) {
	return value, nil
}

type TestGuard struct {
	Name string
}

func (g *TestGuard) CanActivate(ctx *RequestContext) bool {
	return true
}

type TestInterceptor struct {
	Name string
}

func (i *TestInterceptor) Intercept(ctx *RequestContext, next Handler) *Response {
	return next(ctx)
}

type TestExceptionFilter struct {
	Name string
}

func (f *TestExceptionFilter) Catch(exception interface{}, host *RequestContext) *Response {
	return &Response{
		StatusCode: 500,
		Body:       "Internal Server Error",
	}
}