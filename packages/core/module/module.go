package module

import (
	"github.com/healtronlabs/gofasta/packages/core/container"
)

// Module represents a Gofasta module
type Module interface {
	Configure(container *container.DIContainer) error
	Initialize() error
	Cleanup() error
	GetProviders() []interface{}
	GetControllers() []interface{}
	GetImports() []Module
}

// BaseModule provides a default implementation of Module
type BaseModule struct {
	providers   []interface{}
	controllers []interface{}
	imports     []Module
}

// NewBaseModule creates a new base module
func NewBaseModule() *BaseModule {
	return &BaseModule{
		providers:   make([]interface{}, 0),
		controllers: make([]interface{}, 0),
		imports:     make([]Module, 0),
	}
}

// Configure configures the module with the DI container
func (m *BaseModule) Configure(container *container.DIContainer) error {
	// Register providers
	for _, provider := range m.providers {
		// Implementation will depend on reflection and struct tags
		// This is a simplified version
	}
	
	return nil
}

// Initialize initializes the module
func (m *BaseModule) Initialize() error {
	return nil
}

// Cleanup cleans up the module
func (m *BaseModule) Cleanup() error {
	return nil
}

// GetProviders returns the module's providers
func (m *BaseModule) GetProviders() []interface{} {
	return m.providers
}

// GetControllers returns the module's controllers
func (m *BaseModule) GetControllers() []interface{} {
	return m.controllers
}

// GetImports returns the module's imports
func (m *BaseModule) GetImports() []Module {
	return m.imports
}

// AddProvider adds a provider to the module
func (m *BaseModule) AddProvider(provider interface{}) {
	m.providers = append(m.providers, provider)
}

// AddController adds a controller to the module
func (m *BaseModule) AddController(controller interface{}) {
	m.controllers = append(m.controllers, controller)
}

// AddImport adds an import to the module
func (m *BaseModule) AddImport(module Module) {
	m.imports = append(m.imports, module)
}