package core

import (
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
	if typeTag, ok := targetType.Tag().Lookup("controller"); ok {
		decorators["controller"] = append(decorators["controller"], &DecoratorMetadata{
			Type: "controller",
			Parameters: map[string]interface{}{
				"path": typeTag,
			},
		})
	}

	if moduleTag, ok := targetType.Tag().Lookup("module"); ok {
		decorators["module"] = append(decorators["module"], &DecoratorMetadata{
			Type: "module",
			Parameters: map[string]interface{}{
				"name": moduleTag,
			},
		})
	}

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

	return decorators
}

// extractMethodDecorators extracts decorators from method tags (this is conceptual - Go doesn't have method tags)
// In practice, this would be implemented through method naming conventions or external metadata
func extractMethodDecorators(method reflect.Method) map[string][]*DecoratorMetadata {
	decorators := make(map[string][]*DecoratorMetadata)

	// This is a placeholder - in real implementation, we would use:
	// 1. Method naming conventions (Get*, Post*, etc.)
	// 2. External metadata files
	// 3. Build-time code generation
	// 4. Runtime registration through method calls

	methodName := method.Name
	
	// Convention-based route detection
	if strings.HasPrefix(methodName, "Get") {
		decorators["route"] = append(decorators["route"], &DecoratorMetadata{
			Type: "route",
			Parameters: map[string]interface{}{
				"method": "GET",
				"path":   inferPathFromMethodName(methodName),
			},
		})
	} else if strings.HasPrefix(methodName, "Post") {
		decorators["route"] = append(decorators["route"], &DecoratorMetadata{
			Type: "route",
			Parameters: map[string]interface{}{
				"method": "POST",
				"path":   inferPathFromMethodName(methodName),
			},
		})
	} else if strings.HasPrefix(methodName, "Put") {
		decorators["route"] = append(decorators["route"], &DecoratorMetadata{
			Type: "route",
			Parameters: map[string]interface{}{
				"method": "PUT",
				"path":   inferPathFromMethodName(methodName),
			},
		})
	} else if strings.HasPrefix(methodName, "Delete") {
		decorators["route"] = append(decorators["route"], &DecoratorMetadata{
			Type: "route",
			Parameters: map[string]interface{}{
				"method": "DELETE",
				"path":   inferPathFromMethodName(methodName),
			},
		})
	}

	return decorators
}

// parseValidationTag parses validation rules from struct tag
func parseValidationTag(tag string) []string {
	rules := strings.Split(tag, ",")
	for i, rule := range rules {
		rules[i] = strings.TrimSpace(rule)
	}
	return rules
}

// inferPathFromMethodName infers REST path from method name
func inferPathFromMethodName(methodName string) string {
	// Remove HTTP method prefix
	var path string
	if strings.HasPrefix(methodName, "Get") {
		path = strings.TrimPrefix(methodName, "Get")
	} else if strings.HasPrefix(methodName, "Post") {
		path = strings.TrimPrefix(methodName, "Post")
	} else if strings.HasPrefix(methodName, "Put") {
		path = strings.TrimPrefix(methodName, "Put")
	} else if strings.HasPrefix(methodName, "Delete") {
		path = strings.TrimPrefix(methodName, "Delete")
	} else {
		return "/"
	}

	// Convert PascalCase to kebab-case path
	if path == "" {
		return "/"
	}

	// Simple conversion: GetUser -> /user, GetUserById -> /user-by-id
	result := strings.ToLower(string(path[0]))
	for i := 1; i < len(path); i++ {
		if path[i] >= 'A' && path[i] <= 'Z' {
			result += "-" + strings.ToLower(string(path[i]))
		} else {
			result += string(path[i])
		}
	}

	return "/" + result
}

// RouteMetadata represents metadata for HTTP routes
type RouteMetadata struct {
	Method     string
	Path       string
	Handler    reflect.Method
	Guards     []Guard
	Pipes      []Pipe
	Interceptors []Interceptor
	Parameters map[string]string
}

// ExtractRouteMetadata extracts route metadata from a controller
func ExtractRouteMetadata(controller interface{}) ([]*RouteMetadata, error) {
	routes := make([]*RouteMetadata, 0)
	
	controllerType := reflect.TypeOf(controller)
	if controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	// Extract controller-level path
	controllerPath := "/"
	if pathTag, ok := controllerType.Tag().Lookup("controller"); ok {
		controllerPath = pathTag
	}

	// Extract route metadata from methods
	for i := 0; i < controllerType.NumMethod(); i++ {
		method := controllerType.Method(i)
		
		// Check if method is a route handler (public method)
		if method.PkgPath == "" { // Public method
			decorators := extractMethodDecorators(method)
			
			if routeDecorators, exists := decorators["route"]; exists {
				for _, decorator := range routeDecorators {
					if httpMethod, ok := decorator.Parameters["method"].(string); ok {
						methodPath := decorator.Parameters["path"].(string)
						fullPath := controllerPath + methodPath
						
						routes = append(routes, &RouteMetadata{
							Method:  httpMethod,
							Path:    fullPath,
							Handler: method,
						})
					}
				}
			}
		}
	}

	return routes, nil
}