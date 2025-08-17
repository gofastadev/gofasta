package core

import (
	"fmt"
	"reflect"
	"strings"
)

// Module defines the interface for Gofasta modules
type Module interface {
	Configure(container *DIContainer) error
	GetProviders() []interface{}
	GetControllers() []interface{}
	GetImports() []Module
	Initialize() error
	Cleanup() error
}

// BaseModule provides a default implementation of Module
type BaseModule struct {
	providers   []interface{}
	controllers []interface{}
	imports     []Module
	metadata    *ModuleMetadata
}

// NewBaseModule creates a new BaseModule
func NewBaseModule() *BaseModule {
	return &BaseModule{
		providers:   make([]interface{}, 0),
		controllers: make([]interface{}, 0),
		imports:     make([]Module, 0),
	}
}

// Configure implements Module interface
func (m *BaseModule) Configure(container *DIContainer) error {
	// Register providers
	for _, provider := range m.providers {
		if err := container.RegisterProvider(provider); err != nil {
			return err
		}
	}

	// Register controllers
	for _, controller := range m.controllers {
		if err := container.RegisterController(controller); err != nil {
			return err
		}
	}

	// Configure imported modules
	for _, importedModule := range m.imports {
		if err := importedModule.Configure(container); err != nil {
			return err
		}
	}

	return nil
}

// GetProviders implements Module interface
func (m *BaseModule) GetProviders() []interface{} {
	return m.providers
}

// GetControllers implements Module interface
func (m *BaseModule) GetControllers() []interface{} {
	return m.controllers
}

// GetImports implements Module interface
func (m *BaseModule) GetImports() []Module {
	return m.imports
}

// Initialize implements Module interface
func (m *BaseModule) Initialize() error {
	// Initialize imported modules first
	for _, importedModule := range m.imports {
		if err := importedModule.Initialize(); err != nil {
			return err
		}
	}
	return nil
}

// Cleanup implements Module interface
func (m *BaseModule) Cleanup() error {
	// Cleanup in reverse order
	for i := len(m.imports) - 1; i >= 0; i-- {
		if err := m.imports[i].Cleanup(); err != nil {
			return err
		}
	}
	return nil
}

// AddProvider adds a provider to the module
func (m *BaseModule) AddProvider(provider interface{}) {
	m.providers = append(m.providers, provider)
}

// AddController adds a controller to the module
func (m *BaseModule) AddController(controller interface{}) {
	m.controllers = append(m.controllers, controller)
}

// AddImport adds an imported module
func (m *BaseModule) AddImport(module Module) {
	m.imports = append(m.imports, module)
}

// SetMetadata sets the module metadata
func (m *BaseModule) SetMetadata(metadata *ModuleMetadata) {
	m.metadata = metadata
}

// GetMetadata gets the module metadata
func (m *BaseModule) GetMetadata() *ModuleMetadata {
	return m.metadata
}

// ModuleBuilder provides a fluent interface for building modules
type ModuleBuilder struct {
	providers   []interface{}
	controllers []interface{}
	imports     []Module
	exports     []interface{}
	metadata    *ModuleMetadata
}

// NewModuleBuilder creates a new module builder
func NewModuleBuilder() *ModuleBuilder {
	return &ModuleBuilder{
		providers:   make([]interface{}, 0),
		controllers: make([]interface{}, 0),
		imports:     make([]Module, 0),
		exports:     make([]interface{}, 0),
	}
}

// WithProviders adds providers to the module
func (b *ModuleBuilder) WithProviders(providers ...interface{}) *ModuleBuilder {
	b.providers = append(b.providers, providers...)
	return b
}

// WithControllers adds controllers to the module
func (b *ModuleBuilder) WithControllers(controllers ...interface{}) *ModuleBuilder {
	b.controllers = append(b.controllers, controllers...)
	return b
}

// WithImports adds imported modules
func (b *ModuleBuilder) WithImports(imports ...Module) *ModuleBuilder {
	b.imports = append(b.imports, imports...)
	return b
}

// WithExports adds exported services
func (b *ModuleBuilder) WithExports(exports ...interface{}) *ModuleBuilder {
	b.exports = append(b.exports, exports...)
	return b
}

// WithMetadata sets the module metadata
func (b *ModuleBuilder) WithMetadata(metadata *ModuleMetadata) *ModuleBuilder {
	b.metadata = metadata
	return b
}

// Build creates the module from the builder configuration
func (b *ModuleBuilder) Build() Module {
	return &DecoratedModule{
		providers:   b.providers,
		controllers: b.controllers,
		imports:     b.imports,
		exports:     b.exports,
		metadata:    b.metadata,
	}
}

// DecoratedModule is a module implementation that supports decorators and metadata
type DecoratedModule struct {
	providers   []interface{}
	controllers []interface{}
	imports     []Module
	exports     []interface{}
	metadata    *ModuleMetadata
}

// Configure configures the module with the DI container
func (m *DecoratedModule) Configure(container *DIContainer) error {
	// Register providers
	for _, provider := range m.providers {
		if err := container.RegisterProvider(provider); err != nil {
			return fmt.Errorf("failed to register provider: %w", err)
		}
	}
	
	// Register controllers
	for _, controller := range m.controllers {
		if err := container.RegisterController(controller); err != nil {
			return fmt.Errorf("failed to register controller: %w", err)
		}
	}
	
	// Configure imported modules
	for _, importedModule := range m.imports {
		if err := importedModule.Configure(container); err != nil {
			return fmt.Errorf("failed to configure imported module: %w", err)
		}
	}
	
	return nil
}

// Initialize initializes the module
func (m *DecoratedModule) Initialize() error {
	// Initialize imported modules first
	for _, importedModule := range m.imports {
		if err := importedModule.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize imported module: %w", err)
		}
	}
	
	return nil
}

// Cleanup cleans up the module
func (m *DecoratedModule) Cleanup() error {
	// Cleanup imported modules
	for _, importedModule := range m.imports {
		if err := importedModule.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup imported module: %w", err)
		}
	}
	
	return nil
}

// GetProviders returns the module's providers
func (m *DecoratedModule) GetProviders() []interface{} {
	return m.providers
}

// GetControllers returns the module's controllers
func (m *DecoratedModule) GetControllers() []interface{} {
	return m.controllers
}

// GetImports returns the module's imports
func (m *DecoratedModule) GetImports() []Module {
	return m.imports
}

// GetExports returns the module's exports
func (m *DecoratedModule) GetExports() []interface{} {
	return m.exports
}

// GetMetadata returns the module's metadata
func (m *DecoratedModule) GetMetadata() *ModuleMetadata {
	return m.metadata
}

// NewDecoratedModule creates a new decorated module from an instance
func NewDecoratedModule(instance interface{}) (*DecoratedModule, error) {
	// Extract metadata from the instance
	metadata, err := ExtractModuleMetadata(instance)
	if err != nil {
		return nil, err
	}
	
	decoratedModule := &DecoratedModule{
		providers:   make([]interface{}, 0),
		controllers: make([]interface{}, 0),
		imports:     make([]Module, 0),
		exports:     make([]interface{}, 0),
		metadata:    metadata,
	}
	
	// Configure the module based on metadata
	if err := decoratedModule.configureFromMetadata(instance); err != nil {
		return nil, err
	}
	
	return decoratedModule, nil
}

// configureFromMetadata configures the module based on extracted metadata
func (m *DecoratedModule) configureFromMetadata(instance interface{}) error {
	if m.metadata == nil {
		return nil
	}
	
	// This would typically involve resolving string names to actual types
	// For now, we'll implement a basic version
	
	instanceType := reflect.TypeOf(instance)
	if instanceType.Kind() == reflect.Ptr {
		instanceType = instanceType.Elem()
	}
	
	// Look for provider and controller fields in the instance
	instanceValue := reflect.ValueOf(instance)
	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}
	
	for i := 0; i < instanceType.NumField(); i++ {
		field := instanceType.Field(i)
		fieldValue := instanceValue.Field(i)
		
		// Check if field is a provider
		if strings.Contains(strings.ToLower(field.Name), "provider") || 
		   strings.Contains(strings.ToLower(field.Name), "service") {
			if fieldValue.IsValid() && !fieldValue.IsNil() {
				m.providers = append(m.providers, fieldValue.Interface())
			}
		}
		
		// Check if field is a controller
		if strings.Contains(strings.ToLower(field.Name), "controller") {
			if fieldValue.IsValid() && !fieldValue.IsNil() {
				m.controllers = append(m.controllers, fieldValue.Interface())
			}
		}
		
		// Check if field is an imported module
		if strings.Contains(strings.ToLower(field.Name), "import") ||
		   strings.Contains(strings.ToLower(field.Name), "module") {
			if fieldValue.IsValid() && !fieldValue.IsNil() {
				if module, ok := fieldValue.Interface().(Module); ok {
					m.imports = append(m.imports, module)
				}
			}
		}
	}
	
	return nil
}