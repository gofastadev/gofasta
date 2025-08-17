package core

import (
	"reflect"
)

// Module defines the interface for Gofasta modules
type Module interface {
	Configure(container *DIContainer) error
	GetProviders() []Provider
	GetControllers() []Controller
	GetImports() []Module
}

// BaseModule provides a default implementation of Module
type BaseModule struct {
	providers   []Provider
	controllers []Controller
	imports     []Module
}

// NewBaseModule creates a new BaseModule
func NewBaseModule() *BaseModule {
	return &BaseModule{
		providers:   make([]Provider, 0),
		controllers: make([]Controller, 0),
		imports:     make([]Module, 0),
	}
}

// Configure implements Module interface
func (m *BaseModule) Configure(container *DIContainer) error {
	// Default implementation - can be overridden by specific modules
	return nil
}

// GetProviders implements Module interface
func (m *BaseModule) GetProviders() []Provider {
	return m.providers
}

// GetControllers implements Module interface
func (m *BaseModule) GetControllers() []Controller {
	return m.controllers
}

// GetImports implements Module interface
func (m *BaseModule) GetImports() []Module {
	return m.imports
}

// AddProvider adds a provider to the module
func (m *BaseModule) AddProvider(provider Provider) {
	m.providers = append(m.providers, provider)
}

// AddController adds a controller to the module
func (m *BaseModule) AddController(controller Controller) {
	m.controllers = append(m.controllers, controller)
}

// AddImport adds an imported module
func (m *BaseModule) AddImport(module Module) {
	m.imports = append(m.imports, module)
}

// ModuleMetadata represents metadata for a module extracted from struct tags
type ModuleMetadata struct {
	Name        string
	Controllers []reflect.Type
	Providers   []reflect.Type
	Imports     []reflect.Type
	Exports     []reflect.Type
}

// ExtractModuleMetadata extracts module metadata from struct tags
func ExtractModuleMetadata(moduleInstance interface{}) (*ModuleMetadata, error) {
	moduleType := reflect.TypeOf(moduleInstance)
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	metadata := &ModuleMetadata{
		Controllers: make([]reflect.Type, 0),
		Providers:   make([]reflect.Type, 0),
		Imports:     make([]reflect.Type, 0),
		Exports:     make([]reflect.Type, 0),
	}

	// Extract module tag
	if moduleTag, ok := moduleType.Tag().Lookup("module"); ok {
		metadata.Name = moduleTag
	}

	// Extract metadata from struct fields and their tags
	for i := 0; i < moduleType.NumField(); i++ {
		field := moduleType.Field(i)

		// Check for controllers tag
		if controllersTag, ok := field.Tag.Lookup("controllers"); ok {
			// Parse controllers list (this is simplified - real implementation would parse comma-separated list)
			_ = controllersTag
		}

		// Check for providers tag
		if providersTag, ok := field.Tag.Lookup("providers"); ok {
			// Parse providers list
			_ = providersTag
		}

		// Check for imports tag
		if importsTag, ok := field.Tag.Lookup("imports"); ok {
			// Parse imports list
			_ = importsTag
		}

		// Check for exports tag
		if exportsTag, ok := field.Tag.Lookup("exports"); ok {
			// Parse exports list
			_ = exportsTag
		}
	}

	return metadata, nil
}
