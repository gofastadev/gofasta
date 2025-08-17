package container

import (
	"fmt"
	"reflect"
	"sync"
)

// Lifecycle defines the lifecycle of a service
type Lifecycle int

const (
	Singleton Lifecycle = iota
	Transient
	Scoped
)

// Provider represents a service provider
type Provider struct {
	Type      reflect.Type
	Factory   func() interface{}
	Instance  interface{}
	Lifecycle Lifecycle
}

// DIContainer is the dependency injection container
type DIContainer struct {
	providers map[reflect.Type]*Provider
	instances map[reflect.Type]interface{}
	mutex     sync.RWMutex
}

// NewDIContainer creates a new DI container
func NewDIContainer() *DIContainer {
	return &DIContainer{
		providers: make(map[reflect.Type]*Provider),
		instances: make(map[reflect.Type]interface{}),
	}
}

// Register registers a provider in the container
func (c *DIContainer) Register(serviceType reflect.Type, factory func() interface{}, lifecycle Lifecycle) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.providers[serviceType] = &Provider{
		Type:      serviceType,
		Factory:   factory,
		Lifecycle: lifecycle,
	}
}

// RegisterInstance registers a singleton instance
func (c *DIContainer) RegisterInstance(serviceType reflect.Type, instance interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.providers[serviceType] = &Provider{
		Type:      serviceType,
		Instance:  instance,
		Lifecycle: Singleton,
	}
	c.instances[serviceType] = instance
}

// Resolve resolves a service from the container
func (c *DIContainer) Resolve(serviceType reflect.Type) (interface{}, error) {
	c.mutex.RLock()
	provider, exists := c.providers[serviceType]
	c.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service of type %s not registered", serviceType.Name())
	}

	switch provider.Lifecycle {
	case Singleton:
		return c.resolveSingleton(serviceType, provider)
	case Transient:
		return c.resolveTransient(provider)
	default:
		return nil, fmt.Errorf("unsupported lifecycle: %v", provider.Lifecycle)
	}
}

func (c *DIContainer) resolveSingleton(serviceType reflect.Type, provider *Provider) (interface{}, error) {
	c.mutex.RLock()
	instance, exists := c.instances[serviceType]
	c.mutex.RUnlock()

	if exists {
		return instance, nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Double-check pattern
	if instance, exists := c.instances[serviceType]; exists {
		return instance, nil
	}

	if provider.Instance != nil {
		c.instances[serviceType] = provider.Instance
		return provider.Instance, nil
	}

	if provider.Factory != nil {
		instance := provider.Factory()
		c.instances[serviceType] = instance
		return instance, nil
	}

	return nil, fmt.Errorf("no factory or instance provided for %s", serviceType.Name())
}

func (c *DIContainer) resolveTransient(provider *Provider) (interface{}, error) {
	if provider.Factory == nil {
		return nil, fmt.Errorf("no factory provided for transient service")
	}
	return provider.Factory(), nil
}

// InjectDependencies injects dependencies into a struct using reflection
func (c *DIContainer) InjectDependencies(target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	structValue := targetValue.Elem()
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Check for inject tag
		if tag := fieldType.Tag.Get("inject"); tag != "" {
			if !field.CanSet() {
				continue
			}

			// Resolve the dependency
			dependency, err := c.Resolve(field.Type())
			if err != nil {
				return fmt.Errorf("failed to inject dependency %s: %w", fieldType.Name, err)
			}

			field.Set(reflect.ValueOf(dependency))
		}
	}

	return nil
}