package core

import (
	"strings"
	
	"github.com/healtronlabs/gofasta/packages/core/decorators"
)

// Re-export types from decorators subpackage for backward compatibility
type DecoratorMetadata = decorators.DecoratorMetadata

// Re-export functions from decorators subpackage for backward compatibility
func ExtractDecorators(target interface{}) (map[string][]*decorators.DecoratorMetadata, error) {
	return decorators.ExtractDecorators(target)
}

// Re-export types from decorators subpackage for backward compatibility
type ControllerMetadata = decorators.ControllerMetadata
type RouteMetadata = decorators.RouteMetadata

// ExtractControllerMetadata extracts controller metadata using Gofasta-style decorators
func ExtractControllerMetadata(controller interface{}) (*ControllerMetadata, error) {
	return decorators.ExtractControllerMetadata(controller)
}

// Re-export additional types and functions
type ModuleMetadata = decorators.ModuleMetadata
type ServiceProviderMetadata = decorators.ServiceProviderMetadata

func ExtractModuleMetadata(moduleInstance interface{}) (*ModuleMetadata, error) {
	return decorators.ExtractModuleMetadata(moduleInstance)
}

func ExtractServiceProviderMetadata(provider interface{}) (*ServiceProviderMetadata, error) {
	return decorators.ExtractServiceProviderMetadata(provider)
}

func ExtractRouteMetadata(controller interface{}, methodName string) (*RouteMetadata, error) {
	return decorators.ExtractRouteMetadata(controller, methodName)
}

func ExtractAllRouteMetadata(controller interface{}) ([]*RouteMetadata, error) {
	return decorators.ExtractAllRouteMetadata(controller)
}

// parseValidationTag is kept for backward compatibility with existing tests
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
