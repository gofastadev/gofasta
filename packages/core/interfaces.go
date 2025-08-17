package core

import (
	"context"
	"net/http"
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

// Parameter decorator interfaces
type Param struct {
	Name string
}

type Query struct {
	Name string
}

type Body struct{}

type Headers struct{}

// Validation decorator interfaces
type IsNotEmpty struct{}

type IsEmail struct{}

type IsString struct{}

type IsNumber struct{}

type IsArray struct{}

type IsObject struct{}

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

// Auth decorator interfaces
type RequireRoles struct {
	Roles []string
}

type Public struct{}

type Auth struct {
	Strategies []string
}