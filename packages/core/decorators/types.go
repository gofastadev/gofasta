package decorators

import (
	"reflect"
)

// DecoratorMetadata represents metadata extracted from struct tags and decorators
type DecoratorMetadata struct {
	Type       string
	Parameters map[string]interface{}
}

// ControllerMetadata represents metadata for a controller
type ControllerMetadata struct {
	Name       string
	Prefix     string
	Path       string   // Backward compatibility
	Middleware []string
	Routes     []*RouteMetadata
	Guards     []string
	Version    string
}

// RouteMetadata represents metadata for a route
type RouteMetadata struct {
	Method     string
	Path       string
	Handler    string
	Guards     []string
	Pipes      []string
	Middleware []string
	Filters    []string
	Validation string
	StatusCode int
}

// ModuleMetadata represents metadata for a module extracted from struct tags
type ModuleMetadata struct {
	Name        string
	Controllers []string
	Providers   []string
	Imports     []string
	Exports     []string
}

// ServiceProviderMetadata represents metadata for a service provider
type ServiceProviderMetadata struct {
	Name         string
	Type         reflect.Type
	Scope        ServiceScope
	Dependencies []string
	Factory      func() (interface{}, error)
}

// ServiceScope represents the lifecycle scope of a service
type ServiceScope int

const (
	ScopeSingleton ServiceScope = iota
	ScopeTransient
	ScopeScoped
)

// Advanced decorator metadata structures for Gofasta framework
type RouteDecoratorMetadata struct {
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Middleware []string          `json:"middleware,omitempty"`
	Guards     []string          `json:"guards,omitempty"`
	Pipes      []string          `json:"pipes,omitempty"`
	Filters    []string          `json:"filters,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	StatusCode int               `json:"statusCode,omitempty"`
}

type ControllerDecoratorMetadata struct {
	Prefix     string   `json:"prefix"`
	Middleware []string `json:"middleware,omitempty"`
	Guards     []string `json:"guards,omitempty"`
	Version    string   `json:"version,omitempty"`
}

type ParameterDecoratorMetadata struct {
	Type       string `json:"type"`       // body, param, query, header
	Name       string `json:"name"`       // parameter name
	Required   bool   `json:"required"`   // is required
	Validation string `json:"validation"` // validation rules
}