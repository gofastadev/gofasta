package core

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestRequestContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")
	
	ctx := context.Background()
	params := map[string]string{"id": "123", "name": "test"}
	query := map[string]string{"param1": "value1", "param2": "value2"}
	headers := map[string]string{"Authorization": "Bearer token123"}
	
	requestCtx := &RequestContext{
		Request:        req,
		Context:        ctx,
		Params:         params,
		Query:          query,
		Headers:        headers,
		Body:           map[string]interface{}{"test": "data"},
		User:           map[string]interface{}{"id": 1, "name": "testuser"},
		Scope:          NewScopedContext(),
	}
	
	// Test GetHeader
	authHeader := requestCtx.GetHeader("Authorization")
	if authHeader != "Bearer token123" {
		t.Errorf("Expected Authorization header 'Bearer token123', got %s", authHeader)
	}
	
	// Test GetParam
	idParam := requestCtx.GetParam("id")
	if idParam != "123" {
		t.Errorf("Expected param id '123', got %s", idParam)
	}
	
	// Test GetQuery
	param1 := requestCtx.GetQuery("param1")
	if param1 != "value1" {
		t.Errorf("Expected query param1 'value1', got %s", param1)
	}
}

func TestResponse(t *testing.T) {
	response := &Response{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       map[string]interface{}{"message": "success"},
	}
	
	if response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}
	
	if response.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", response.Headers["Content-Type"])
	}
	
	body, ok := response.Body.(map[string]interface{})
	if !ok {
		t.Error("Response body is not of expected type")
	} else {
		if body["message"] != "success" {
			t.Errorf("Expected body message 'success', got %v", body["message"])
		}
	}
}

func TestBaseServiceProvider(t *testing.T) {
	factory := func() interface{} {
		return &TestLogger{Level: "DEBUG"}
	}
	
	provider := NewServiceProvider("test-logger", factory, ScopeSingleton)
	
	if provider == nil {
		t.Fatal("NewServiceProvider() returned nil")
	}
	
	if provider.GetName() != "test-logger" {
		t.Errorf("Expected name 'test-logger', got %s", provider.GetName())
	}
	
	if provider.GetScope() != ScopeSingleton {
		t.Errorf("Expected scope ScopeSingleton, got %v", provider.GetScope())
	}
	
	instance := provider.Provide()
	logger, ok := instance.(*TestLogger)
	if !ok {
		t.Error("Provider did not return expected type")
	} else {
		if logger.Level != "DEBUG" {
			t.Errorf("Expected logger level 'DEBUG', got %s", logger.Level)
		}
	}
	
	// Test dependencies
	if len(provider.GetDependencies()) != 0 {
		t.Errorf("Expected 0 dependencies initially, got %d", len(provider.GetDependencies()))
	}
	
	// Add dependency
	dependencyType := reflect.TypeOf((*TestConfig)(nil)).Elem()
	provider.AddDependency(dependencyType)
	
	if len(provider.GetDependencies()) != 1 {
		t.Errorf("Expected 1 dependency after adding, got %d", len(provider.GetDependencies()))
	}
	
	if provider.GetDependencies()[0] != dependencyType {
		t.Error("Dependency not added correctly")
	}
}

func TestFactoryProvider(t *testing.T) {
	factory := func() interface{} {
		return &TestConfig{DatabaseURL: "test://localhost"}
	}
	
	provider := FactoryProvider("test-config", factory, ScopeTransient)
	
	if provider.GetName() != "test-config" {
		t.Errorf("Expected name 'test-config', got %s", provider.GetName())
	}
	
	if provider.GetScope() != ScopeTransient {
		t.Errorf("Expected scope ScopeTransient, got %v", provider.GetScope())
	}
	
	instance := provider.Provide()
	config, ok := instance.(*TestConfig)
	if !ok {
		t.Error("Factory provider did not return expected type")
	} else {
		if config.DatabaseURL != "test://localhost" {
			t.Errorf("Expected database URL 'test://localhost', got %s", config.DatabaseURL)
		}
	}
}

func TestValueProvider(t *testing.T) {
	config := &TestConfig{
		DatabaseURL: "prod://database",
		APIKey:      "secret-key",
	}
	
	provider := ValueProvider("prod-config", config)
	
	if provider.GetName() != "prod-config" {
		t.Errorf("Expected name 'prod-config', got %s", provider.GetName())
	}
	
	if provider.GetScope() != ScopeSingleton {
		t.Errorf("Expected scope ScopeSingleton for value provider, got %v", provider.GetScope())
	}
	
	instance := provider.Provide()
	if instance != config {
		t.Error("Value provider did not return the same instance")
	}
}

func TestClassProvider(t *testing.T) {
	classType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	provider := ClassProvider("logger-class", classType, ScopeTransient)
	
	if provider.GetName() != "logger-class" {
		t.Errorf("Expected name 'logger-class', got %s", provider.GetName())
	}
	
	if provider.GetScope() != ScopeTransient {
		t.Errorf("Expected scope ScopeTransient, got %v", provider.GetScope())
	}
	
	instance := provider.Provide()
	_, ok := instance.(*TestLogger)
	if !ok {
		t.Error("Class provider did not return expected type")
	}
}

func TestExecutionContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	requestCtx := &RequestContext{Request: req}
	
	handler := func(ctx *RequestContext) *Response {
		return &Response{StatusCode: 200}
	}
	
	classType := reflect.TypeOf((*TestController1)(nil)).Elem()
	method, _ := classType.MethodByName("GetUsers")
	args := []interface{}{"arg1", "arg2"}
	metadata := map[string]interface{}{"key": "value"}
	
	app := CreateApp(&TestAppModule{})
	
	execCtx := &ExecutionContext{
		Request:     requestCtx,
		Handler:     handler,
		Class:       classType,
		Method:      method,
		Args:        args,
		Metadata:    metadata,
		Application: app,
	}
	
	if execCtx.GetRequest() != requestCtx {
		t.Error("GetRequest() did not return correct request context")
	}
	
	if execCtx.GetHandler() == nil {
		t.Error("GetHandler() returned nil")
	}
	
	if execCtx.GetClass() != classType {
		t.Error("GetClass() did not return correct class type")
	}
	
	if execCtx.GetMethod().Name != "GetUsers" {
		t.Errorf("Expected method name 'GetUsers', got %s", execCtx.GetMethod().Name)
	}
	
	if len(execCtx.GetArgs()) != 2 {
		t.Errorf("Expected 2 args, got %d", len(execCtx.GetArgs()))
	}
	
	if execCtx.GetMetadata()["key"] != "value" {
		t.Errorf("Expected metadata key 'value', got %v", execCtx.GetMetadata()["key"])
	}
	
	if execCtx.GetApplication() != app {
		t.Error("GetApplication() did not return correct application")
	}
}

func TestArgumentsHost(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	requestCtx := &RequestContext{Request: req}
	
	response := &Response{StatusCode: 200}
	
	handler := func(ctx *RequestContext) *Response {
		return response
	}
	
	argsHost := &ArgumentsHost{
		Request:  requestCtx,
		Response: response,
		Next:     handler,
	}
	
	if argsHost.GetRequest() != requestCtx {
		t.Error("GetRequest() did not return correct request context")
	}
	
	if argsHost.GetResponse() != response {
		t.Error("GetResponse() did not return correct response")
	}
	
	if argsHost.GetNext() == nil {
		t.Error("GetNext() returned nil")
	}
}

func TestDefaultCallHandler(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	requestCtx := &RequestContext{Request: req}
	
	expectedResponse := &Response{StatusCode: 200, Body: "test"}
	handler := func(ctx *RequestContext) *Response {
		return expectedResponse
	}
	
	callHandler := NewCallHandler(handler, requestCtx)
	
	if callHandler == nil {
		t.Fatal("NewCallHandler() returned nil")
	}
	
	response := callHandler.Handle()
	if response != expectedResponse {
		t.Error("CallHandler.Handle() did not return expected response")
	}
}

func TestDefaultModuleRef(t *testing.T) {
	container := NewDIContainer()
	ctx := context.Background()
	
	// Register a service
	container.RegisterProvider(&TestLogger{})
	container.RegisterInstance(reflect.TypeOf((*TestConfig)(nil)).Elem(), &TestConfig{DatabaseURL: "test"})
	container.namedServices["test-config"] = container.services[reflect.TypeOf((*TestConfig)(nil)).Elem()]
	container.Initialize()
	
	moduleRef := NewModuleRef(container, ctx)
	
	if moduleRef == nil {
		t.Fatal("NewModuleRef() returned nil")
	}
	
	// Test Get
	loggerType := reflect.TypeOf((*TestLogger)(nil)).Elem()
	instance, err := moduleRef.Get(loggerType)
	if err != nil {
		t.Fatalf("ModuleRef.Get() failed: %v", err)
	}
	
	_, ok := instance.(*TestLogger)
	if !ok {
		t.Error("ModuleRef.Get() did not return expected type")
	}
	
	// Test GetByName
	instance, err = moduleRef.GetByName("test-config")
	if err != nil {
		t.Fatalf("ModuleRef.GetByName() failed: %v", err)
	}
	
	_, ok = instance.(*TestConfig)
	if !ok {
		t.Error("ModuleRef.GetByName() did not return expected type")
	}
	
	// Test Create
	instance, err = moduleRef.Create(loggerType)
	if err != nil {
		t.Fatalf("ModuleRef.Create() failed: %v", err)
	}
	
	_, ok = instance.(*TestLogger)
	if !ok {
		t.Error("ModuleRef.Create() did not return expected type")
	}
	
	// Test Resolve
	instance, err = moduleRef.Resolve(loggerType)
	if err != nil {
		t.Fatalf("ModuleRef.Resolve() failed: %v", err)
	}
	
	_, ok = instance.(*TestLogger)
	if !ok {
		t.Error("ModuleRef.Resolve() did not return expected type")
	}
}

func TestServiceScope_Constants(t *testing.T) {
	if ScopeSingleton != 0 {
		t.Errorf("Expected ScopeSingleton to be 0, got %d", ScopeSingleton)
	}
	
	if ScopeTransient != 1 {
		t.Errorf("Expected ScopeTransient to be 1, got %d", ScopeTransient)
	}
	
	if ScopeScoped != 2 {
		t.Errorf("Expected ScopeScoped to be 2, got %d", ScopeScoped)
	}
}

func TestLifecyclePhase_Constants(t *testing.T) {
	if PhaseBeforeCreate != 0 {
		t.Errorf("Expected PhaseBeforeCreate to be 0, got %d", PhaseBeforeCreate)
	}
	
	if PhaseAfterCreate != 1 {
		t.Errorf("Expected PhaseAfterCreate to be 1, got %d", PhaseAfterCreate)
	}
	
	if PhaseBeforeDestroy != 2 {
		t.Errorf("Expected PhaseBeforeDestroy to be 2, got %d", PhaseBeforeDestroy)
	}
	
	if PhaseAfterDestroy != 3 {
		t.Errorf("Expected PhaseAfterDestroy to be 3, got %d", PhaseAfterDestroy)
	}
}

func TestResolveOptions(t *testing.T) {
	options := &ResolveOptions{
		Strict: true,
		Scope:  "request",
	}
	
	if !options.Strict {
		t.Error("Expected Strict to be true")
	}
	
	if options.Scope != "request" {
		t.Errorf("Expected Scope to be 'request', got %s", options.Scope)
	}
}

// Test interfaces implementation
func TestInterfaceImplementations(t *testing.T) {
	// Test that TestPipe implements Pipe
	var pipe Pipe = &TestPipe{}
	if pipe == nil {
		t.Error("TestPipe does not implement Pipe interface")
	}
	
	// Test that TestGuard implements Guard
	var guard Guard = &TestGuard{}
	if guard == nil {
		t.Error("TestGuard does not implement Guard interface")
	}
	
	// Test that TestInterceptor implements Interceptor
	var interceptor Interceptor = &TestInterceptor{}
	if interceptor == nil {
		t.Error("TestInterceptor does not implement Interceptor interface")
	}
	
	// Test that TestExceptionFilter implements ExceptionFilter
	var filter ExceptionFilter = &TestExceptionFilter{}
	if filter == nil {
		t.Error("TestExceptionFilter does not implement ExceptionFilter interface")
	}
	
	// Test that BaseServiceProvider implements ServiceProvider
	var provider ServiceProvider = NewServiceProvider("test", func() interface{} { return nil }, ScopeSingleton)
	if provider == nil {
		t.Error("BaseServiceProvider does not implement ServiceProvider interface")
	}
}