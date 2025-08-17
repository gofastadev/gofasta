package core

import (
	"fmt"
	"reflect"
	"sync"
)

// ServiceScope defines the lifecycle of a service
type ServiceScope int

const (
	ScopeSingleton ServiceScope = iota
	ScopeTransient
	ScopeScoped
)

// ServiceDescriptor describes how to create and manage a service
type ServiceDescriptor struct {
	ServiceType  reflect.Type
	Factory      func() (interface{}, error)
	Instance     interface{}
	Scope        ServiceScope
	Dependencies []reflect.Type
}

// DIContainer is the dependency injection container
type DIContainer struct {
	services    map[reflect.Type]*ServiceDescriptor
	instances   map[reflect.Type]interface{}
	mutex       sync.RWMutex
	initialized bool
}

// NewDIContainer creates a new DI container
func NewDIContainer() *DIContainer {
	return &DIContainer{
		services:  make(map[reflect.Type]*ServiceDescriptor),
		instances: make(map[reflect.Type]interface{}),
	}
}

// RegisterProvider registers a provider with the container
func (c *DIContainer) RegisterProvider(provider Provider) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.initialized {
		return fmt.Errorf("cannot register providers after container is initialized")
	}

	providerType := reflect.TypeOf(provider)
	if providerType.Kind() == reflect.Ptr {
		providerType = providerType.Elem()
	}

	// Extract dependencies from struct tags
	dependencies := c.extractDependencies(providerType)

	descriptor := &ServiceDescriptor{
		ServiceType:  providerType,
		Factory:      func() (interface{}, error) { return c.createInstance(providerType) },
		Scope:        ScopeSingleton, // Default scope
		Dependencies: dependencies,
	}

	c.services[providerType] = descriptor
	return nil
}

// RegisterController registers a controller with the container
func (c *DIContainer) RegisterController(controller Controller) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.initialized {
		return fmt.Errorf("cannot register controllers after container is initialized")
	}

	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	// Extract dependencies from struct tags
	dependencies := c.extractDependencies(controllerType)

	descriptor := &ServiceDescriptor{
		ServiceType:  controllerType,
		Factory:      func() (interface{}, error) { return c.createInstance(controllerType) },
		Scope:        ScopeSingleton, // Controllers are typically singletons
		Dependencies: dependencies,
	}

	c.services[controllerType] = descriptor
	return nil
}

// Initialize initializes all registered services and resolves dependencies
func (c *DIContainer) Initialize() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.initialized {
		return nil
	}

	// Check for circular dependencies
	if err := c.checkCircularDependencies(); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	// Initialize singleton services
	for serviceType, descriptor := range c.services {
		if descriptor.Scope == ScopeSingleton {
			if _, err := c.resolveDependencies(serviceType, make(map[reflect.Type]bool)); err != nil {
				return fmt.Errorf("failed to initialize service %s: %w", serviceType.Name(), err)
			}
		}
	}

	c.initialized = true
	return nil
}

// Resolve resolves a service from the container
func (c *DIContainer) Resolve(serviceType reflect.Type) (interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.initialized {
		return nil, fmt.Errorf("container not initialized")
	}

	return c.resolveDependencies(serviceType, make(map[reflect.Type]bool))
}

// resolveDependencies resolves a service and its dependencies recursively
func (c *DIContainer) resolveDependencies(serviceType reflect.Type, resolving map[reflect.Type]bool) (interface{}, error) {
	if resolving[serviceType] {
		return nil, fmt.Errorf("circular dependency detected for service %s", serviceType.Name())
	}

	descriptor, exists := c.services[serviceType]
	if !exists {
		return nil, fmt.Errorf("service %s not registered", serviceType.Name())
	}

	// Check if singleton instance already exists
	if descriptor.Scope == ScopeSingleton {
		if instance, exists := c.instances[serviceType]; exists {
			return instance, nil
		}
	}

	resolving[serviceType] = true
	defer delete(resolving, serviceType)

	// Create the instance
	instance, err := c.createInstanceWithDependencies(serviceType, resolving)
	if err != nil {
		return nil, err
	}

	// Store singleton instance
	if descriptor.Scope == ScopeSingleton {
		c.instances[serviceType] = instance
	}

	return instance, nil
}

// createInstanceWithDependencies creates an instance and injects its dependencies
func (c *DIContainer) createInstanceWithDependencies(serviceType reflect.Type, resolving map[reflect.Type]bool) (interface{}, error) {
	// Create instance
	instance := reflect.New(serviceType).Interface()

	// Inject dependencies
	if err := c.injectDependencies(instance, resolving); err != nil {
		return nil, fmt.Errorf("failed to inject dependencies: %w", err)
	}

	return instance, nil
}

// injectDependencies injects dependencies into a service instance
func (c *DIContainer) injectDependencies(instance interface{}, resolving map[reflect.Type]bool) error {
	instanceValue := reflect.ValueOf(instance)
	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}

	instanceType := instanceValue.Type()

	for i := 0; i < instanceType.NumField(); i++ {
		field := instanceType.Field(i)
		fieldValue := instanceValue.Field(i)

		// Check for inject tag
		if injectTag, ok := field.Tag.Lookup("inject"); ok {
			if !fieldValue.CanSet() {
				continue
			}

			var dependencyType reflect.Type
			if field.Type.Kind() == reflect.Ptr {
				dependencyType = field.Type.Elem()
			} else {
				dependencyType = field.Type
			}

			// Resolve dependency
			dependency, err := c.resolveDependencies(dependencyType, resolving)
			if err != nil {
				// If named injection is specified, try to resolve by name
				if injectTag != "" {
					// TODO: Implement named dependency resolution
				}
				return fmt.Errorf("failed to resolve dependency %s: %w", dependencyType.Name(), err)
			}

			// Set the field value
			dependencyValue := reflect.ValueOf(dependency)
			if field.Type.Kind() == reflect.Ptr && dependencyValue.Kind() != reflect.Ptr {
				// If field expects pointer but dependency is value, get address
				if dependencyValue.CanAddr() {
					dependencyValue = dependencyValue.Addr()
				}
			}

			fieldValue.Set(dependencyValue)
		}
	}

	return nil
}

// extractDependencies extracts dependency types from struct tags
func (c *DIContainer) extractDependencies(serviceType reflect.Type) []reflect.Type {
	dependencies := make([]reflect.Type, 0)

	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		if _, ok := field.Tag.Lookup("inject"); ok {
			var dependencyType reflect.Type
			if field.Type.Kind() == reflect.Ptr {
				dependencyType = field.Type.Elem()
			} else {
				dependencyType = field.Type
			}
			dependencies = append(dependencies, dependencyType)
		}
	}

	return dependencies
}

// createInstance creates a basic instance of the given type
func (c *DIContainer) createInstance(serviceType reflect.Type) (interface{}, error) {
	return reflect.New(serviceType).Interface(), nil
}

// checkCircularDependencies checks for circular dependencies in the service graph
func (c *DIContainer) checkCircularDependencies() error {
	for serviceType := range c.services {
		visited := make(map[reflect.Type]bool)
		recursionStack := make(map[reflect.Type]bool)
		
		if c.hasCycle(serviceType, visited, recursionStack) {
			return fmt.Errorf("circular dependency starting from %s", serviceType.Name())
		}
	}
	return nil
}

// hasCycle performs DFS to detect cycles in the dependency graph
func (c *DIContainer) hasCycle(serviceType reflect.Type, visited, recursionStack map[reflect.Type]bool) bool {
	visited[serviceType] = true
	recursionStack[serviceType] = true

	descriptor, exists := c.services[serviceType]
	if !exists {
		return false
	}

	for _, dependency := range descriptor.Dependencies {
		if !visited[dependency] {
			if c.hasCycle(dependency, visited, recursionStack) {
				return true
			}
		} else if recursionStack[dependency] {
			return true
		}
	}

	recursionStack[serviceType] = false
	return false
}