package core

import (
	"fmt"
	"reflect"
	"strings"
)

// DecoratorMetadata represents metadata extracted from struct tags and decorators
type DecoratorMetadata struct {
	Type       string
	Parameters map[string]interface{}
}

// ExtractDecorators extracts decorator metadata from struct tags
func ExtractDecorators(target interface{}) (map[string][]*DecoratorMetadata, error) {
	decorators := make(map[string][]*DecoratorMetadata)

	targetType := reflect.TypeOf(target)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// Extract type-level decorators (from struct tags)
	// Note: In Go, struct types don't have tags, only fields do
	// We'll look for special fields or use a different approach for type-level metadata

	// Extract field-level decorators
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldDecorators := extractFieldDecorators(field)

		for decoratorType, metadata := range fieldDecorators {
			decorators[decoratorType] = append(decorators[decoratorType], metadata...)
		}
	}

	// Extract method-level decorators
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	for i := 0; i < targetType.NumMethod(); i++ {
		method := targetType.Method(i)
		methodDecorators := extractMethodDecorators(method)

		for decoratorType, metadata := range methodDecorators {
			decorators[decoratorType] = append(decorators[decoratorType], metadata...)
		}
	}

	return decorators, nil
}

// extractFieldDecorators extracts decorators from struct field tags
func extractFieldDecorators(field reflect.StructField) map[string][]*DecoratorMetadata {
	decorators := make(map[string][]*DecoratorMetadata)

	// Inject decorator
	if injectTag, ok := field.Tag.Lookup("inject"); ok {
		decorators["inject"] = append(decorators["inject"], &DecoratorMetadata{
			Type: "inject",
			Parameters: map[string]interface{}{
				"name":  injectTag,
				"field": field.Name,
				"type":  field.Type,
			},
		})
	}

	// Validation decorators
	if validateTag, ok := field.Tag.Lookup("validate"); ok {
		validationRules := parseValidationTag(validateTag)
		for _, rule := range validationRules {
			decorators["validate"] = append(decorators["validate"], &DecoratorMetadata{
				Type: "validate",
				Parameters: map[string]interface{}{
					"rule":  rule,
					"field": field.Name,
				},
			})
		}
	}

	// JSON binding
	if jsonTag, ok := field.Tag.Lookup("json"); ok {
		decorators["json"] = append(decorators["json"], &DecoratorMetadata{
			Type: "json",
			Parameters: map[string]interface{}{
				"name":  jsonTag,
				"field": field.Name,
			},
		})
	}

	// Controller decorator (for embedded controller metadata)
	if controllerTag, ok := field.Tag.Lookup("controller"); ok {
		decorators["controller"] = append(decorators["controller"], &DecoratorMetadata{
			Type: "controller",
			Parameters: map[string]interface{}{
				"path":  controllerTag,
				"field": field.Name,
			},
		})
	}

	// Module decorator (for embedded module metadata)
	if moduleTag, ok := field.Tag.Lookup("module"); ok {
		decorators["module"] = append(decorators["module"], &DecoratorMetadata{
			Type: "module",
			Parameters: map[string]interface{}{
				"name":  moduleTag,
				"field": field.Name,
			},
		})
	}

	return decorators
}

// extractMethodDecorators extracts decorators from method metadata
// Note: Go doesn't support method decorators directly, so this is a placeholder
// for future implementation using code generation or other techniques
func extractMethodDecorators(method reflect.Method) map[string][]*DecoratorMetadata {
	decorators := make(map[string][]*DecoratorMetadata)

	// This would be implemented using code generation or reflection on method names
	// For now, we'll return empty decorators

	return decorators
}

// parseValidationTag parses validation tag into individual rules
func parseValidationTag(tag string) []string {
	if tag == "" {
		return nil
	}

	rules := strings.Split(tag, ",")
	for i, rule := range rules {
		rules[i] = strings.TrimSpace(rule)
	}

	return rules
}

// ControllerMetadata represents metadata for a controller
type ControllerMetadata struct {
	Path       string
	Middleware []string
	Routes     []*RouteMetadata
}

// RouteMetadata represents metadata for a route
type RouteMetadata struct {
	Method     string
	Path       string
	Handler    string
	Guards     []string
	Pipes      []string
	Validation string
}

// ExtractControllerMetadata extracts controller metadata from struct tags and methods
func ExtractControllerMetadata(controller interface{}) (*ControllerMetadata, error) {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	metadata := &ControllerMetadata{
		Routes: make([]*RouteMetadata, 0),
	}

	// Look for controller metadata in struct fields
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)

		if controllerTag, ok := field.Tag.Lookup("controller"); ok {
			metadata.Path = controllerTag
		}

		if middlewareTag, ok := field.Tag.Lookup("middleware"); ok {
			metadata.Middleware = strings.Split(middlewareTag, ",")
			for j, mw := range metadata.Middleware {
				metadata.Middleware[j] = strings.TrimSpace(mw)
			}
		}
	}

	// Extract route metadata from methods
	// This would typically be done through code generation or build-time analysis
	// For now, we'll use naming conventions or other mechanisms

	return metadata, nil
}

// ModuleMetadata represents metadata for a module extracted from struct tags
type ModuleMetadata struct {
	Name        string
	Controllers []string
	Providers   []string
	Imports     []string
	Exports     []string
}

// ExtractModuleMetadata extracts module metadata from struct tags
func ExtractModuleMetadata(moduleInstance interface{}) (*ModuleMetadata, error) {
	moduleType := reflect.TypeOf(moduleInstance)
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	metadata := &ModuleMetadata{
		Controllers: make([]string, 0),
		Providers:   make([]string, 0),
		Imports:     make([]string, 0),
		Exports:     make([]string, 0),
	}

	// Extract metadata from struct fields and their tags
	for i := 0; i < moduleType.NumField(); i++ {
		field := moduleType.Field(i)

		// Check for module name
		if moduleTag, ok := field.Tag.Lookup("module"); ok {
			metadata.Name = moduleTag
		}

		// Check for controllers tag
		if controllersTag, ok := field.Tag.Lookup("controllers"); ok {
			controllers := strings.Split(controllersTag, ",")
			for _, controller := range controllers {
				metadata.Controllers = append(metadata.Controllers, strings.TrimSpace(controller))
			}
		}

		// Check for providers tag
		if providersTag, ok := field.Tag.Lookup("providers"); ok {
			providers := strings.Split(providersTag, ",")
			for _, provider := range providers {
				metadata.Providers = append(metadata.Providers, strings.TrimSpace(provider))
			}
		}

		// Check for imports tag
		if importsTag, ok := field.Tag.Lookup("imports"); ok {
			imports := strings.Split(importsTag, ",")
			for _, imp := range imports {
				metadata.Imports = append(metadata.Imports, strings.TrimSpace(imp))
			}
		}

		// Check for exports tag
		if exportsTag, ok := field.Tag.Lookup("exports"); ok {
			exports := strings.Split(exportsTag, ",")
			for _, exp := range exports {
				metadata.Exports = append(metadata.Exports, strings.TrimSpace(exp))
			}
		}
	}

	return metadata, nil
}

// ServiceProviderMetadata represents metadata for a service provider
type ServiceProviderMetadata struct {
	Name         string
	Type         reflect.Type
	Scope        ServiceScope
	Dependencies []string
	Factory      func() (interface{}, error)
}

// ExtractServiceProviderMetadata extracts service provider metadata
func ExtractServiceProviderMetadata(provider interface{}) (*ServiceProviderMetadata, error) {
	providerType := reflect.TypeOf(provider)
	if providerType.Kind() == reflect.Ptr {
		providerType = providerType.Elem()
	}

	metadata := &ServiceProviderMetadata{
		Type:         providerType,
		Scope:        ScopeSingleton, // Default scope
		Dependencies: make([]string, 0),
	}

	// Extract metadata from struct fields
	for i := 0; i < providerType.NumField(); i++ {
		field := providerType.Field(i)

		// Check for service name
		if serviceTag, ok := field.Tag.Lookup("service"); ok {
			metadata.Name = serviceTag
		}

		// Check for scope
		if scopeTag, ok := field.Tag.Lookup("scope"); ok {
			switch scopeTag {
			case "singleton":
				metadata.Scope = ScopeSingleton
			case "transient":
				metadata.Scope = ScopeTransient
			case "scoped":
				metadata.Scope = ScopeScoped
			}
		}

		// Check for dependencies
		if _, ok := field.Tag.Lookup("inject"); ok {
			metadata.Dependencies = append(metadata.Dependencies, field.Name)
		}
	}

	return metadata, nil
}

// ExtractRouteMetadata extracts route metadata from a controller method
func ExtractRouteMetadata(controller interface{}, methodName string) (*RouteMetadata, error) {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	// Find the method
	method, found := controllerType.MethodByName(methodName)
	if !found {
		return nil, fmt.Errorf("method %s not found in controller %s", methodName, controllerType.Name())
	}

	metadata := &RouteMetadata{
		Handler: methodName,
		Guards:  make([]string, 0),
		Pipes:   make([]string, 0),
	}

	// For now, we'll use naming conventions to determine HTTP method and path
	// In a full implementation, this would use struct tags or code generation
	methodNameLower := strings.ToLower(methodName)

	if strings.HasPrefix(methodNameLower, "get") {
		metadata.Method = "GET"
		metadata.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Get"))
	} else if strings.HasPrefix(methodNameLower, "post") {
		metadata.Method = "POST"
		metadata.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Post"))
	} else if strings.HasPrefix(methodNameLower, "put") {
		metadata.Method = "PUT"
		metadata.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Put"))
	} else if strings.HasPrefix(methodNameLower, "delete") {
		metadata.Method = "DELETE"
		metadata.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Delete"))
	} else if strings.HasPrefix(methodNameLower, "patch") {
		metadata.Method = "PATCH"
		metadata.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Patch"))
	} else {
		// Default to GET
		metadata.Method = "GET"
		metadata.Path = "/" + strings.ToLower(methodName)
	}

	// Extract additional metadata from method signature or other sources
	// This is a simplified implementation
	_ = method // Avoid unused variable

	return metadata, nil
}

// ExtractAllRouteMetadata extracts all route metadata from a controller
func ExtractAllRouteMetadata(controller interface{}) ([]*RouteMetadata, error) {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	routes := make([]*RouteMetadata, 0)

	// Iterate through all methods
	for i := 0; i < controllerType.NumMethod(); i++ {
		method := controllerType.Method(i)

		// Skip private methods and common methods
		if !method.IsExported() ||
			method.Name == "String" ||
			method.Name == "Error" ||
			method.Name == "GoString" {
			continue
		}

		routeMetadata, err := ExtractRouteMetadata(controller, method.Name)
		if err != nil {
			continue // Skip methods that can't be converted to routes
		}

		routes = append(routes, routeMetadata)
	}

	return routes, nil
}
