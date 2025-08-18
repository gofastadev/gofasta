package decorators

import (
	"reflect"
	"sync"
)

// ControllerRegistry manages registered controller metadata
type ControllerRegistry struct {
	mu          sync.RWMutex
	controllers map[reflect.Type]*ControllerDecoratorMetadata
	routes      map[reflect.Type]map[string]*RouteDecoratorMetadata
}

var globalRegistry = &ControllerRegistry{
	controllers: make(map[reflect.Type]*ControllerDecoratorMetadata),
	routes:      make(map[reflect.Type]map[string]*RouteDecoratorMetadata),
}

// RegisterController registers controller metadata programmatically
func RegisterController(controllerType reflect.Type, prefix string, middleware ...string) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.controllers[controllerType] = &ControllerDecoratorMetadata{
		Prefix:     prefix,
		Middleware: middleware,
	}

	if globalRegistry.routes[controllerType] == nil {
		globalRegistry.routes[controllerType] = make(map[string]*RouteDecoratorMetadata)
	}
}

// RegisterRoute registers route metadata programmatically
func RegisterRoute(controllerType reflect.Type, methodName, httpMethod, path string, middleware []string, guards []string, statusCode int) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if globalRegistry.routes[controllerType] == nil {
		globalRegistry.routes[controllerType] = make(map[string]*RouteDecoratorMetadata)
	}

	globalRegistry.routes[controllerType][methodName] = &RouteDecoratorMetadata{
		Method:     httpMethod,
		Path:       path,
		Middleware: middleware,
		Guards:     guards,
		StatusCode: statusCode,
	}
}

// GetControllerMetadata retrieves registered controller metadata
func GetControllerMetadata(controllerType reflect.Type) (*ControllerDecoratorMetadata, map[string]*RouteDecoratorMetadata, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	controllerMeta, hasController := globalRegistry.controllers[controllerType]
	routesMeta, hasRoutes := globalRegistry.routes[controllerType]

	return controllerMeta, routesMeta, hasController && hasRoutes
}

// ControllerBuilder provides a fluent API for building controller metadata
type ControllerBuilder struct {
	controllerType reflect.Type
	prefix         string
	middleware     []string
	guards         []string
	version        string
}

// Controller creates a new controller builder
func Controller(prefix string) *ControllerBuilder {
	return &ControllerBuilder{
		prefix:     prefix,
		middleware: []string{},
		guards:     []string{},
	}
}

// UseMiddleware adds middleware to the controller
func (cb *ControllerBuilder) UseMiddleware(middleware ...string) *ControllerBuilder {
	cb.middleware = append(cb.middleware, middleware...)
	return cb
}

// UseGuards adds guards to the controller
func (cb *ControllerBuilder) UseGuards(guards ...string) *ControllerBuilder {
	cb.guards = append(cb.guards, guards...)
	return cb
}

// Version sets the API version
func (cb *ControllerBuilder) Version(version string) *ControllerBuilder {
	cb.version = version
	return cb
}

// Register registers the controller with the given type
func (cb *ControllerBuilder) Register(controllerInstance interface{}) *RouteBuilder {
	cb.controllerType = reflect.TypeOf(controllerInstance)
	if cb.controllerType.Kind() == reflect.Ptr {
		cb.controllerType = cb.controllerType.Elem()
	}

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.controllers[cb.controllerType] = &ControllerDecoratorMetadata{
		Prefix:     cb.prefix,
		Middleware: cb.middleware,
		Guards:     cb.guards,
		Version:    cb.version,
	}

	if globalRegistry.routes[cb.controllerType] == nil {
		globalRegistry.routes[cb.controllerType] = make(map[string]*RouteDecoratorMetadata)
	}

	return &RouteBuilder{
		controllerType: cb.controllerType,
	}
}

// RouteBuilder provides a fluent API for building route metadata
type RouteBuilder struct {
	controllerType reflect.Type
}

// Route adds a route to the controller
func (rb *RouteBuilder) Route(methodName, httpMethod, path string) *RouteMethodBuilder {
	return &RouteMethodBuilder{
		controllerType: rb.controllerType,
		methodName:     methodName,
		httpMethod:     httpMethod,
		path:           path,
		middleware:     []string{},
		guards:         []string{},
		pipes:          []string{},
		filters:        []string{},
	}
}

// Get adds a GET route
func (rb *RouteBuilder) Get(methodName, path string) *RouteMethodBuilder {
	return rb.Route(methodName, "GET", path)
}

// Post adds a POST route
func (rb *RouteBuilder) Post(methodName, path string) *RouteMethodBuilder {
	return rb.Route(methodName, "POST", path)
}

// Put adds a PUT route
func (rb *RouteBuilder) Put(methodName, path string) *RouteMethodBuilder {
	return rb.Route(methodName, "PUT", path)
}

// Delete adds a DELETE route
func (rb *RouteBuilder) Delete(methodName, path string) *RouteMethodBuilder {
	return rb.Route(methodName, "DELETE", path)
}

// Patch adds a PATCH route
func (rb *RouteBuilder) Patch(methodName, path string) *RouteMethodBuilder {
	return rb.Route(methodName, "PATCH", path)
}

// RouteMethodBuilder provides a fluent API for configuring individual routes
type RouteMethodBuilder struct {
	controllerType reflect.Type
	methodName     string
	httpMethod     string
	path           string
	middleware     []string
	guards         []string
	pipes          []string
	filters        []string
	statusCode     int
}

// UseMiddleware adds middleware to the route
func (rmb *RouteMethodBuilder) UseMiddleware(middleware ...string) *RouteMethodBuilder {
	rmb.middleware = append(rmb.middleware, middleware...)
	return rmb
}

// UseGuards adds guards to the route
func (rmb *RouteMethodBuilder) UseGuards(guards ...string) *RouteMethodBuilder {
	rmb.guards = append(rmb.guards, guards...)
	return rmb
}

// UsePipes adds pipes to the route
func (rmb *RouteMethodBuilder) UsePipes(pipes ...string) *RouteMethodBuilder {
	rmb.pipes = append(rmb.pipes, pipes...)
	return rmb
}

// UseFilters adds filters to the route
func (rmb *RouteMethodBuilder) UseFilters(filters ...string) *RouteMethodBuilder {
	rmb.filters = append(rmb.filters, filters...)
	return rmb
}

// HttpCode sets the status code for the route
func (rmb *RouteMethodBuilder) HttpCode(statusCode int) *RouteMethodBuilder {
	rmb.statusCode = statusCode
	return rmb
}

// Register registers the route
func (rmb *RouteMethodBuilder) Register() *RouteBuilder {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if globalRegistry.routes[rmb.controllerType] == nil {
		globalRegistry.routes[rmb.controllerType] = make(map[string]*RouteDecoratorMetadata)
	}

	globalRegistry.routes[rmb.controllerType][rmb.methodName] = &RouteDecoratorMetadata{
		Method:     rmb.httpMethod,
		Path:       rmb.path,
		Middleware: rmb.middleware,
		Guards:     rmb.guards,
		Pipes:      rmb.pipes,
		Filters:    rmb.filters,
		StatusCode: rmb.statusCode,
	}

	return &RouteBuilder{
		controllerType: rmb.controllerType,
	}
}

// ClearRegistry clears all registered metadata (useful for testing)
func ClearRegistry() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.controllers = make(map[reflect.Type]*ControllerDecoratorMetadata)
	globalRegistry.routes = make(map[reflect.Type]map[string]*RouteDecoratorMetadata)
}