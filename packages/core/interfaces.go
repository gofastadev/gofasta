package core

import (
	"context"
	"net/http"
	"reflect"
)

// Provider represents a service provider that can be registered with the DI container
type Provider interface{}

// Controller represents a controller that handles HTTP requests
type Controller interface{}

// RequestContext represents the context of an HTTP request
type RequestContext struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Context        context.Context
	Params         map[string]string
	Query          map[string]string
	Headers        map[string]string
	Body           interface{}
	User           interface{}
	Scope          *ScopedContext
}

// GetHeader gets a header value from the request
func (ctx *RequestContext) GetHeader(key string) string {
	return ctx.Request.Header.Get(key)
}

// GetParam gets a URL parameter value
func (ctx *RequestContext) GetParam(key string) string {
	return ctx.Params[key]
}

// GetQuery gets a query parameter value
func (ctx *RequestContext) GetQuery(key string) string {
	return ctx.Query[key]
}

// SetHeader sets a response header
func (ctx *RequestContext) SetHeader(key, value string) {
	ctx.ResponseWriter.Header().Set(key, value)
}

// SetStatus sets the response status code
func (ctx *RequestContext) SetStatus(code int) {
	ctx.ResponseWriter.WriteHeader(code)
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       interface{}
}

// Guard interface for authentication and authorization
type Guard interface {
	CanActivate(ctx *RequestContext) bool
}

// Pipe interface for request/response transformation
type Pipe interface {
	Transform(value interface{}, metadata *PipeMetadata) (interface{}, error)
}

// PipeMetadata contains metadata about the pipe transformation
type PipeMetadata struct {
	Type     string
	Target   interface{}
	Metatype interface{}
	Data     map[string]interface{}
}

// Interceptor interface for request/response interception
type Interceptor interface {
	Intercept(ctx *RequestContext, next Handler) *Response
}

// Handler represents a request handler function
type Handler func(ctx *RequestContext) *Response

// ExceptionFilter interface for handling exceptions
type ExceptionFilter interface {
	Catch(exception interface{}, host *RequestContext) *Response
}

// MiddlewareFunc represents a middleware function
type MiddlewareFunc func(ctx *RequestContext, next Handler) *Response

// Injectable marker interface for dependency injection
type Injectable interface {
	IsInjectable() bool
}

// Initializable interface for services that need initialization
type Initializable interface {
	Initialize() error
}

// Cleanupable interface for services that need cleanup
type Cleanupable interface {
	Cleanup() error
}

// HealthCheckable interface for services that provide health checks
type HealthCheckable interface {
	HealthCheck() error
}

// Configurable interface for services that can be configured
type Configurable interface {
	Configure(config map[string]interface{}) error
}

// UseGuards decorator interface
type UseGuards struct {
	Guards []interface{}
}

// UsePipes decorator interface
type UsePipes struct {
	Pipes []interface{}
}

// UseInterceptors decorator interface
type UseInterceptors struct {
	Interceptors []interface{}
}

// UseFilters decorator interface
type UseFilters struct {
	Filters []interface{}
}

// Module decorator interface
type ModuleDecorator struct {
	Controllers []interface{}
	Providers   []interface{}
	Imports     []interface{}
	Exports     []interface{}
}

// Controller decorator interface
type ControllerDecorator struct {
	Path       string
	Middleware []interface{}
}

// Service decorator interface
type ServiceDecorator struct {
	Name  string
	Scope ServiceScope
}

// Route decorator interfaces
type Get struct {
	Path string
}

type Post struct {
	Path string
}

type Put struct {
	Path string
}

type Delete struct {
	Path string
}

type Patch struct {
	Path string
}

type Options struct {
	Path string
}

type Head struct {
	Path string
}

// Parameter decorator interfaces
type Param struct {
	Name string
}

type Query struct {
	Name string
}

type Body struct{}

type Headers struct{}

type Req struct{}

type Res struct{}

type Session struct{}

type Cookies struct{}

// Validation decorator interfaces
type IsNotEmpty struct{}

type IsEmail struct{}

type IsString struct{}

type IsNumber struct{}

type IsArray struct{}

type IsObject struct{}

type IsBoolean struct{}

type IsDate struct{}

type IsUUID struct{}

type IsURL struct{}

type IsIP struct{}

type Length struct {
	Min int
	Max int
}

type Min struct {
	Value int
}

type Max struct {
	Value int
}

type MinLength struct {
	Value int
}

type MaxLength struct {
	Value int
}

type Matches struct {
	Pattern string
}

type IsIn struct {
	Values []interface{}
}

type IsNotIn struct {
	Values []interface{}
}

// Auth decorator interfaces
type RequireRoles struct {
	Roles []string
}

type RequirePermissions struct {
	Permissions []string
}

type Public struct{}

type Auth struct {
	Strategies []string
}

// Cache decorator interfaces
type Cache struct {
	TTL int
	Key string
}

type CacheEvict struct {
	Key string
}

// Rate limiting decorator interfaces
type RateLimit struct {
	Limit  int
	Window int
}

// Timeout decorator interface
type Timeout struct {
	Duration int
}

// Retry decorator interface
type Retry struct {
	Attempts int
	Delay    int
}

// ServiceProvider interface for advanced service configuration
type ServiceProvider interface {
	Provide() interface{}
	GetScope() ServiceScope
	GetName() string
	GetDependencies() []reflect.Type
}

// BaseServiceProvider provides a default implementation of ServiceProvider
type BaseServiceProvider struct {
	name         string
	scope        ServiceScope
	dependencies []reflect.Type
	factory      func() interface{}
}

// NewServiceProvider creates a new service provider
func NewServiceProvider(name string, factory func() interface{}, scope ServiceScope) *BaseServiceProvider {
	return &BaseServiceProvider{
		name:         name,
		factory:      factory,
		scope:        scope,
		dependencies: make([]reflect.Type, 0),
	}
}

// Provide implements ServiceProvider interface
func (sp *BaseServiceProvider) Provide() interface{} {
	return sp.factory()
}

// GetScope implements ServiceProvider interface
func (sp *BaseServiceProvider) GetScope() ServiceScope {
	return sp.scope
}

// GetName implements ServiceProvider interface
func (sp *BaseServiceProvider) GetName() string {
	return sp.name
}

// GetDependencies implements ServiceProvider interface
func (sp *BaseServiceProvider) GetDependencies() []reflect.Type {
	return sp.dependencies
}

// AddDependency adds a dependency to the service provider
func (sp *BaseServiceProvider) AddDependency(dependencyType reflect.Type) {
	sp.dependencies = append(sp.dependencies, dependencyType)
}

// FactoryProvider creates a service provider from a factory function
func FactoryProvider(name string, factory func() interface{}, scope ServiceScope) ServiceProvider {
	return NewServiceProvider(name, factory, scope)
}

// ValueProvider creates a service provider from a value
func ValueProvider(name string, value interface{}) ServiceProvider {
	return NewServiceProvider(name, func() interface{} { return value }, ScopeSingleton)
}

// ClassProvider creates a service provider from a class type
func ClassProvider(name string, classType reflect.Type, scope ServiceScope) ServiceProvider {
	return NewServiceProvider(name, func() interface{} {
		return reflect.New(classType).Interface()
	}, scope)
}

// ExecutionContext represents the execution context for a request
type ExecutionContext struct {
	Request     *RequestContext
	Handler     Handler
	Class       reflect.Type
	Method      reflect.Method
	Args        []interface{}
	Metadata    map[string]interface{}
	Application Application
}

// GetRequest returns the request context
func (ec *ExecutionContext) GetRequest() *RequestContext {
	return ec.Request
}

// GetHandler returns the handler
func (ec *ExecutionContext) GetHandler() Handler {
	return ec.Handler
}

// GetClass returns the class type
func (ec *ExecutionContext) GetClass() reflect.Type {
	return ec.Class
}

// GetMethod returns the method
func (ec *ExecutionContext) GetMethod() reflect.Method {
	return ec.Method
}

// GetArgs returns the arguments
func (ec *ExecutionContext) GetArgs() []interface{} {
	return ec.Args
}

// GetMetadata returns the metadata
func (ec *ExecutionContext) GetMetadata() map[string]interface{} {
	return ec.Metadata
}

// GetApplication returns the application instance
func (ec *ExecutionContext) GetApplication() Application {
	return ec.Application
}

// ArgumentsHost represents the arguments host for exception filters
type ArgumentsHost struct {
	Request  *RequestContext
	Response *Response
	Next     Handler
}

// GetRequest returns the request context
func (ah *ArgumentsHost) GetRequest() *RequestContext {
	return ah.Request
}

// GetResponse returns the response
func (ah *ArgumentsHost) GetResponse() *Response {
	return ah.Response
}

// GetNext returns the next handler
func (ah *ArgumentsHost) GetNext() Handler {
	return ah.Next
}

// CallHandler represents a call handler for interceptors
type CallHandler interface {
	Handle() *Response
}

// DefaultCallHandler is the default implementation of CallHandler
type DefaultCallHandler struct {
	handler Handler
	context *RequestContext
}

// NewCallHandler creates a new call handler
func NewCallHandler(handler Handler, context *RequestContext) CallHandler {
	return &DefaultCallHandler{
		handler: handler,
		context: context,
	}
}

// Handle implements CallHandler interface
func (ch *DefaultCallHandler) Handle() *Response {
	return ch.handler(ch.context)
}

// ModuleRef represents a reference to a module for dynamic module loading
type ModuleRef interface {
	Get(serviceType reflect.Type) (interface{}, error)
	GetByName(name string) (interface{}, error)
	Create(serviceType reflect.Type) (interface{}, error)
	Resolve(serviceType reflect.Type, options ...ResolveOptions) (interface{}, error)
}

// ResolveOptions represents options for service resolution
type ResolveOptions struct {
	Strict bool
	Scope  string
}

// DefaultModuleRef is the default implementation of ModuleRef
type DefaultModuleRef struct {
	container *DIContainer
	context   context.Context
}

// NewModuleRef creates a new module reference
func NewModuleRef(container *DIContainer, context context.Context) ModuleRef {
	return &DefaultModuleRef{
		container: container,
		context:   context,
	}
}

// Get implements ModuleRef interface
func (mr *DefaultModuleRef) Get(serviceType reflect.Type) (interface{}, error) {
	return mr.container.ResolveWithContext(mr.context, serviceType)
}

// GetByName implements ModuleRef interface
func (mr *DefaultModuleRef) GetByName(name string) (interface{}, error) {
	return mr.container.ResolveNamedWithContext(mr.context, name)
}

// Create implements ModuleRef interface
func (mr *DefaultModuleRef) Create(serviceType reflect.Type) (interface{}, error) {
	// Create a new instance without using the container's cache
	return reflect.New(serviceType).Interface(), nil
}

// Resolve implements ModuleRef interface
func (mr *DefaultModuleRef) Resolve(serviceType reflect.Type, options ...ResolveOptions) (interface{}, error) {
	// For now, just delegate to Get
	// In a full implementation, this would handle the options
	return mr.Get(serviceType)
}