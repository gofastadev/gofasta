package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/healtronlabs/gofasta/packages/core/decorators"
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
	Factory      func(ctx context.Context, container *DIContainer) (interface{}, error)
	Instance     interface{}
	Scope        ServiceScope
	Dependencies []reflect.Type
	Name         string
	Metadata     *ServiceProviderMetadata
}

// ScopedContext represents a scoped context for dependency resolution
type ScopedContext struct {
	instances map[reflect.Type]interface{}
	mutex     sync.RWMutex
}

// NewScopedContext creates a new scoped context
func NewScopedContext() *ScopedContext {
	return &ScopedContext{
		instances: make(map[reflect.Type]interface{}),
	}
}

// DIContainer is the dependency injection container
type DIContainer struct {
	services         map[reflect.Type]*ServiceDescriptor
	namedServices    map[string]*ServiceDescriptor
	instances        map[reflect.Type]interface{}
	scopedContexts   map[string]*ScopedContext
	mutex            sync.RWMutex
	initialized      bool
	lifecycleHooks   map[reflect.Type][]LifecycleHook
	dependencyGraph  map[reflect.Type][]reflect.Type
	resolutionStack  []reflect.Type
}

// LifecycleHook represents a lifecycle hook for services
type LifecycleHook struct {
	Phase    LifecyclePhase
	Callback func(instance interface{}) error
}

// LifecyclePhase represents different phases in service lifecycle
type LifecyclePhase int

const (
	PhaseBeforeCreate LifecyclePhase = iota
	PhaseAfterCreate
	PhaseBeforeDestroy
	PhaseAfterDestroy
)

// NewDIContainer creates a new DI container
func NewDIContainer() *DIContainer {
	return &DIContainer{
		services:        make(map[reflect.Type]*ServiceDescriptor),
		namedServices:   make(map[string]*ServiceDescriptor),
		instances:       make(map[reflect.Type]interface{}),
		scopedContexts:  make(map[string]*ScopedContext),
		lifecycleHooks:  make(map[reflect.Type][]LifecycleHook),
		dependencyGraph: make(map[reflect.Type][]reflect.Type),
		resolutionStack: make([]reflect.Type, 0),
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

	// Extract metadata from the provider
	metadata, err := ExtractServiceProviderMetadata(provider)
	if err != nil {
		return fmt.Errorf("failed to extract provider metadata: %w", err)
	}

	// Extract dependencies from struct tags
	dependencies := c.extractDependencies(providerType)

	// Convert decorators.ServiceScope to core ServiceScope
	var scope ServiceScope
	switch metadata.Scope {
	case decorators.ScopeSingleton:
		scope = ScopeSingleton
	case decorators.ScopeTransient:
		scope = ScopeTransient
	case decorators.ScopeScoped:
		scope = ScopeScoped
	default:
		scope = ScopeSingleton
	}

	descriptor := &ServiceDescriptor{
		ServiceType:  providerType,
		Factory:      c.createProviderFactory(provider),
		Scope:        scope,
		Dependencies: dependencies,
		Name:         metadata.Name,
		Metadata:     metadata,
	}

	c.services[providerType] = descriptor
	
	// Register by name if provided
	if metadata.Name != "" {
		c.namedServices[metadata.Name] = descriptor
	}

	// Update dependency graph
	c.dependencyGraph[providerType] = dependencies

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
		Factory:      c.createControllerFactory(controller),
		Scope:        ScopeSingleton, // Controllers are typically singletons
		Dependencies: dependencies,
	}

	c.services[controllerType] = descriptor
	
	// Update dependency graph
	c.dependencyGraph[controllerType] = dependencies

	return nil
}

// RegisterService registers a service with custom configuration
func (c *DIContainer) RegisterService(serviceType reflect.Type, factory func(ctx context.Context, container *DIContainer) (interface{}, error), scope ServiceScope, name ...string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.initialized {
		return fmt.Errorf("cannot register services after container is initialized")
	}

	descriptor := &ServiceDescriptor{
		ServiceType: serviceType,
		Factory:     factory,
		Scope:       scope,
	}

	if len(name) > 0 && name[0] != "" {
		descriptor.Name = name[0]
		c.namedServices[name[0]] = descriptor
	}

	c.services[serviceType] = descriptor
	return nil
}

// RegisterInstance registers a singleton instance
func (c *DIContainer) RegisterInstance(serviceType reflect.Type, instance interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.initialized {
		return fmt.Errorf("cannot register instances after container is initialized")
	}

	descriptor := &ServiceDescriptor{
		ServiceType: serviceType,
		Instance:    instance,
		Scope:       ScopeSingleton,
		Factory: func(ctx context.Context, container *DIContainer) (interface{}, error) {
			return instance, nil
		},
	}

	c.services[serviceType] = descriptor
	c.instances[serviceType] = instance
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

	// Initialize singleton services in dependency order
	initOrder, err := c.getInitializationOrder()
	if err != nil {
		return fmt.Errorf("failed to determine initialization order: %w", err)
	}

	ctx := context.Background()
	for _, serviceType := range initOrder {
		descriptor, exists := c.services[serviceType]
		if !exists || descriptor.Scope != ScopeSingleton {
			continue
		}

		if _, err := c.resolveDependencies(ctx, serviceType, nil); err != nil {
			return fmt.Errorf("failed to initialize service %s: %w", serviceType.Name(), err)
		}
	}

	c.initialized = true
	return nil
}

// Resolve resolves a service from the container
func (c *DIContainer) Resolve(serviceType reflect.Type) (interface{}, error) {
	return c.ResolveWithContext(context.Background(), serviceType)
}

// ResolveWithContext resolves a service from the container with context
func (c *DIContainer) ResolveWithContext(ctx context.Context, serviceType reflect.Type) (interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.initialized {
		return nil, fmt.Errorf("container not initialized")
	}

	return c.resolveDependencies(ctx, serviceType, nil)
}

// ResolveNamed resolves a named service from the container
func (c *DIContainer) ResolveNamed(name string) (interface{}, error) {
	return c.ResolveNamedWithContext(context.Background(), name)
}

// ResolveNamedWithContext resolves a named service from the container with context
func (c *DIContainer) ResolveNamedWithContext(ctx context.Context, name string) (interface{}, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.initialized {
		return nil, fmt.Errorf("container not initialized")
	}

	return c.resolveNamedWithContextInternal(ctx, name)
}

// resolveNamedWithContextInternal resolves a named service without acquiring locks (internal use)
func (c *DIContainer) resolveNamedWithContextInternal(ctx context.Context, name string) (interface{}, error) {
	descriptor, exists := c.namedServices[name]
	if !exists {
		return nil, fmt.Errorf("named service %s not registered", name)
	}

	return c.resolveDependencies(ctx, descriptor.ServiceType, nil)
}

// CreateScope creates a new scoped context
func (c *DIContainer) CreateScope(scopeId string) *ScopedContext {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	scope := NewScopedContext()
	c.scopedContexts[scopeId] = scope
	return scope
}

// DestroyScope destroys a scoped context and cleans up scoped instances
func (c *DIContainer) DestroyScope(scopeId string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	scope, exists := c.scopedContexts[scopeId]
	if !exists {
		return nil
	}

	// Cleanup scoped instances
	for serviceType, instance := range scope.instances {
		if err := c.executeLifecycleHooks(serviceType, instance, PhaseBeforeDestroy); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Error executing before destroy hook for %s: %v\n", serviceType.Name(), err)
		}

		// Call cleanup method if it exists
		if cleaner, ok := instance.(interface{ Cleanup() error }); ok {
			if err := cleaner.Cleanup(); err != nil {
				fmt.Printf("Error cleaning up %s: %v\n", serviceType.Name(), err)
			}
		}

		if err := c.executeLifecycleHooks(serviceType, instance, PhaseAfterDestroy); err != nil {
			fmt.Printf("Error executing after destroy hook for %s: %v\n", serviceType.Name(), err)
		}
	}

	delete(c.scopedContexts, scopeId)
	return nil
}

// AddLifecycleHook adds a lifecycle hook for a service type
func (c *DIContainer) AddLifecycleHook(serviceType reflect.Type, phase LifecyclePhase, callback func(instance interface{}) error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hooks := c.lifecycleHooks[serviceType]
	hooks = append(hooks, LifecycleHook{
		Phase:    phase,
		Callback: callback,
	})
	c.lifecycleHooks[serviceType] = hooks
}

// resolveDependencies resolves a service and its dependencies recursively
func (c *DIContainer) resolveDependencies(ctx context.Context, serviceType reflect.Type, scopedContext *ScopedContext) (interface{}, error) {
	// Check for circular dependency in current resolution stack
	for _, stackType := range c.resolutionStack {
		if stackType == serviceType {
			return nil, fmt.Errorf("circular dependency detected for service %s", serviceType.Name())
		}
	}

	descriptor, exists := c.services[serviceType]
	if !exists {
		return nil, fmt.Errorf("service %s not registered", serviceType.Name())
	}

	// Handle different scopes
	switch descriptor.Scope {
	case ScopeSingleton:
		if instance, exists := c.instances[serviceType]; exists {
			return instance, nil
		}
	case ScopeScoped:
		if scopedContext != nil {
			scopedContext.mutex.RLock()
			if instance, exists := scopedContext.instances[serviceType]; exists {
				scopedContext.mutex.RUnlock()
				return instance, nil
			}
			scopedContext.mutex.RUnlock()
		}
	case ScopeTransient:
		// Always create new instance for transient services
	}

	// Add to resolution stack
	c.resolutionStack = append(c.resolutionStack, serviceType)
	defer func() {
		// Remove from resolution stack
		if len(c.resolutionStack) > 0 {
			c.resolutionStack = c.resolutionStack[:len(c.resolutionStack)-1]
		}
	}()

	// Execute before create hooks
	if err := c.executeLifecycleHooks(serviceType, nil, PhaseBeforeCreate); err != nil {
		return nil, fmt.Errorf("before create hook failed for %s: %w", serviceType.Name(), err)
	}

	// Create the instance
	instance, err := descriptor.Factory(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance of %s: %w", serviceType.Name(), err)
	}

	// Inject dependencies
	if err := c.injectDependencies(ctx, instance, scopedContext); err != nil {
		return nil, fmt.Errorf("failed to inject dependencies for %s: %w", serviceType.Name(), err)
	}

	// Execute after create hooks
	if err := c.executeLifecycleHooks(serviceType, instance, PhaseAfterCreate); err != nil {
		return nil, fmt.Errorf("after create hook failed for %s: %w", serviceType.Name(), err)
	}

	// Store instance based on scope
	switch descriptor.Scope {
	case ScopeSingleton:
		c.instances[serviceType] = instance
	case ScopeScoped:
		if scopedContext != nil {
			scopedContext.mutex.Lock()
			scopedContext.instances[serviceType] = instance
			scopedContext.mutex.Unlock()
		}
	}

	return instance, nil
}

// createProviderFactory creates a factory function for a provider
func (c *DIContainer) createProviderFactory(provider Provider) func(ctx context.Context, container *DIContainer) (interface{}, error) {
	return func(ctx context.Context, container *DIContainer) (interface{}, error) {
		providerType := reflect.TypeOf(provider)
		if providerType.Kind() == reflect.Ptr {
			providerType = providerType.Elem()
		}
		return reflect.New(providerType).Interface(), nil
	}
}

// createControllerFactory creates a factory function for a controller
func (c *DIContainer) createControllerFactory(controller Controller) func(ctx context.Context, container *DIContainer) (interface{}, error) {
	return func(ctx context.Context, container *DIContainer) (interface{}, error) {
		controllerType := reflect.TypeOf(controller)
		if controllerType.Kind() == reflect.Ptr {
			controllerType = controllerType.Elem()
		}
		return reflect.New(controllerType).Interface(), nil
	}
}

// injectDependencies injects dependencies into a service instance
func (c *DIContainer) injectDependencies(ctx context.Context, instance interface{}, scopedContext *ScopedContext) error {
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

			var dependency interface{}
			var err error

			// Try named resolution first if tag has a value
			if injectTag != "" {
				dependency, err = c.resolveNamedWithContextInternal(ctx, injectTag)
			} else {
				// Resolve by type
				dependency, err = c.resolveDependencies(ctx, dependencyType, scopedContext)
			}

			if err != nil {
				return fmt.Errorf("failed to resolve dependency %s: %w", dependencyType.Name(), err)
			}

			// Set the field value
			dependencyValue := reflect.ValueOf(dependency)
			if field.Type.Kind() == reflect.Ptr && dependencyValue.Kind() != reflect.Ptr {
				// If field expects pointer but dependency is value, get address
				if dependencyValue.CanAddr() {
					dependencyValue = dependencyValue.Addr()
				}
			} else if field.Type.Kind() != reflect.Ptr && dependencyValue.Kind() == reflect.Ptr {
				// If field expects value but dependency is pointer, dereference
				dependencyValue = dependencyValue.Elem()
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

// checkCircularDependencies checks for circular dependencies in the service graph
func (c *DIContainer) checkCircularDependencies() error {
	visited := make(map[reflect.Type]bool)
	recursionStack := make(map[reflect.Type]bool)

	for serviceType := range c.services {
		if !visited[serviceType] {
			if c.hasCycle(serviceType, visited, recursionStack) {
				return fmt.Errorf("circular dependency detected starting from %s", serviceType.Name())
			}
		}
	}
	return nil
}

// hasCycle performs DFS to detect cycles in the dependency graph
func (c *DIContainer) hasCycle(serviceType reflect.Type, visited, recursionStack map[reflect.Type]bool) bool {
	visited[serviceType] = true
	recursionStack[serviceType] = true

	dependencies, exists := c.dependencyGraph[serviceType]
	if !exists {
		recursionStack[serviceType] = false
		return false
	}

	for _, dependency := range dependencies {
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

// getInitializationOrder returns the order in which services should be initialized
func (c *DIContainer) getInitializationOrder() ([]reflect.Type, error) {
	var order []reflect.Type
	visited := make(map[reflect.Type]bool)
	temp := make(map[reflect.Type]bool)

	var visit func(reflect.Type) error
	visit = func(serviceType reflect.Type) error {
		if temp[serviceType] {
			return fmt.Errorf("circular dependency detected")
		}
		if visited[serviceType] {
			return nil
		}

		temp[serviceType] = true
		dependencies := c.dependencyGraph[serviceType]
		for _, dep := range dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		temp[serviceType] = false
		visited[serviceType] = true
		order = append(order, serviceType)
		return nil
	}

	for serviceType := range c.services {
		if !visited[serviceType] {
			if err := visit(serviceType); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

// executeLifecycleHooks executes lifecycle hooks for a service
func (c *DIContainer) executeLifecycleHooks(serviceType reflect.Type, instance interface{}, phase LifecyclePhase) error {
	hooks, exists := c.lifecycleHooks[serviceType]
	if !exists {
		return nil
	}

	for _, hook := range hooks {
		if hook.Phase == phase {
			if err := hook.Callback(instance); err != nil {
				return err
			}
		}
	}

	return nil
}

// Shutdown gracefully shuts down the container and cleans up resources
func (c *DIContainer) Shutdown(timeout time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Cleanup all scoped contexts
	for scopeId := range c.scopedContexts {
		if err := c.DestroyScope(scopeId); err != nil {
			fmt.Printf("Error destroying scope %s: %v\n", scopeId, err)
		}
	}

	// Cleanup singleton instances in reverse order
	initOrder, _ := c.getInitializationOrder()
	for i := len(initOrder) - 1; i >= 0; i-- {
		serviceType := initOrder[i]
		if instance, exists := c.instances[serviceType]; exists {
			select {
			case <-ctx.Done():
				return fmt.Errorf("shutdown timeout exceeded")
			default:
				if err := c.executeLifecycleHooks(serviceType, instance, PhaseBeforeDestroy); err != nil {
					fmt.Printf("Error executing before destroy hook for %s: %v\n", serviceType.Name(), err)
				}

				if cleaner, ok := instance.(interface{ Cleanup() error }); ok {
					if err := cleaner.Cleanup(); err != nil {
						fmt.Printf("Error cleaning up %s: %v\n", serviceType.Name(), err)
					}
				}

				if err := c.executeLifecycleHooks(serviceType, instance, PhaseAfterDestroy); err != nil {
					fmt.Printf("Error executing after destroy hook for %s: %v\n", serviceType.Name(), err)
				}
			}
		}
	}

	c.initialized = false
	return nil
}