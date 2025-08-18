package decorators

import (
	"fmt"
	"reflect"
	"strings"
)

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