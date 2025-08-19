package decorators

import (
	"fmt"
	"reflect"
	"strings"
)

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

// ExtractControllerMetadata extracts controller metadata using Gofasta-style decorators
func ExtractControllerMetadata(controller interface{}) (*ControllerMetadata, error) {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	// Try to extract modern Gofasta decorators first
	controllerMeta, routesMeta, err := ExtractModernDecorators(controller)
	if err != nil {
		// Fallback to convention-based extraction
		return extractControllerMetadataFallback(controller)
	}

	metadata := &ControllerMetadata{
		Name:       controllerType.Name(),
		Prefix:     controllerMeta.Prefix,
		Path:       controllerMeta.Prefix, // Backward compatibility
		Middleware: controllerMeta.Middleware,
		Guards:     controllerMeta.Guards,
		Version:    controllerMeta.Version,
		Routes:     make([]*RouteMetadata, 0),
	}

	// Convert modern route metadata to internal format
	for methodName, routeMeta := range routesMeta {
		route := &RouteMetadata{
			Method:     routeMeta.Method,
			Path:       BuildFullPath(controllerMeta, routeMeta),
			Handler:    methodName,
			Guards:     GetCombinedGuards(controllerMeta, routeMeta),
			Pipes:      routeMeta.Pipes,
			Middleware: GetCombinedMiddleware(controllerMeta, routeMeta),
			Filters:    routeMeta.Filters,
			StatusCode: routeMeta.StatusCode,
		}
		metadata.Routes = append(metadata.Routes, route)
	}

	return metadata, nil
}

// extractControllerMetadataFallback provides backward compatibility with struct tags
func extractControllerMetadataFallback(controller interface{}) (*ControllerMetadata, error) {
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	metadata := &ControllerMetadata{
		Name:   controllerType.Name(),
		Routes: make([]*RouteMetadata, 0),
	}

	// Look for controller metadata in struct fields (backward compatibility)
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)

		if controllerTag, ok := field.Tag.Lookup("controller"); ok {
			metadata.Path = controllerTag
			metadata.Prefix = controllerTag
		}

		if middlewareTag, ok := field.Tag.Lookup("middleware"); ok {
			metadata.Middleware = strings.Split(middlewareTag, ",")
			for j, mw := range metadata.Middleware {
				metadata.Middleware[j] = strings.TrimSpace(mw)
			}
		}
	}

	// Extract routes using convention-based approach
	routes, err := ExtractAllRouteMetadata(controller)
	if err != nil {
		return nil, fmt.Errorf("failed to extract routes: %w", err)
	}

	metadata.Routes = routes
	return metadata, nil
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