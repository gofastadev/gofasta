package codegen

import (
	"fmt"
	"strings"
)

// generateServiceDeclaration generates Go code for a service
func (g *CodeGenerator) generateServiceDeclaration(service *ServiceDeclaration) error {
	// Generate struct declaration
	g.writeLine(fmt.Sprintf("type %s struct {", service.Name))
	g.indent()

	// Generate fields with injection tags
	for _, field := range service.Fields {
		tag := g.generateInjectionTag(field)
		if tag != "" {
			g.writeLine(fmt.Sprintf("%s %s `%s`", field.Name, field.Type, tag))
		} else {
			g.writeLine(fmt.Sprintf("%s %s", field.Name, field.Type))
		}
	}

	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate Initialize method if Injectable
	if g.hasDecorator(service.Decorators, "Injectable") {
		g.generateServiceInitializeMethod(service)
		g.writeLine("")
	}

	// Generate Provider factory if Injectable or Scope decorator is present
	if g.hasDecorator(service.Decorators, "Injectable") || g.hasDecorator(service.Decorators, "Scope") {
		g.generateProviderFactory(service)
		g.writeLine("")
	}

	// Generate methods
	for _, method := range service.Methods {
		if err := g.generateServiceMethod(service, method); err != nil {
			return err
		}
		g.writeLine("")
	}

	return nil
}

// generateServiceMethod generates a regular service method
func (g *CodeGenerator) generateServiceMethod(service *ServiceDeclaration, method *MethodNode) error {
	// Generate method signature
	signature := g.generateMethodSignature(service.Name, method, false)
	g.writeLine(signature + " {")
	g.indent()

	// Generate method body placeholder
	g.writeLine("// TODO: Implement method logic")
	if method.ReturnType != "" && method.ReturnType != "void" {
		g.writeLine("return nil")
	}

	g.unindent()
	g.writeLine("}")

	return nil
}

// generateServiceInitializeMethod generates Initialize method for services
func (g *CodeGenerator) generateServiceInitializeMethod(service *ServiceDeclaration) {
	g.writeLine(fmt.Sprintf("func (s *%s) Initialize() error {", service.Name))
	g.indent()
	g.writeLine("// TODO: Add initialization logic")
	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
}

// generateProviderFactory generates a provider factory function for injectable services
func (g *CodeGenerator) generateProviderFactory(service *ServiceDeclaration) {
	factoryName := fmt.Sprintf("New%s", service.Name)
	
	// Extract scope from @Injectable decorator
	scope := g.getInjectableScope(service)
	
	// Generate factory function signature
	g.writeLine(fmt.Sprintf("func %s(container *core.DIContainer) (*%s, error) {", factoryName, service.Name))
	g.indent()
	
	// Create instance
	g.writeLine(fmt.Sprintf("instance := &%s{}", service.Name))
	g.writeLine("")
	
	// Generate dependency injection for each field
	for _, field := range service.Fields {
		g.generateFieldDependencyInjection(field)
	}
	
	// Initialize the service if it has an Initialize method
	g.writeLine("")
	g.writeLine("if initializer, ok := interface{}(instance).(interface{ Initialize() error }); ok {")
	g.indent()
	g.writeLine("if err := initializer.Initialize(); err != nil {")
	g.indent()
	g.writeLine("return nil, err")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	
	g.writeLine("")
	g.writeLine("return instance, nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	
	// Generate provider registration helper
	g.generateProviderRegistration(service, factoryName, scope)
}

// generateProviderRegistration generates provider registration helper function
func (g *CodeGenerator) generateProviderRegistration(service *ServiceDeclaration, factoryName, scope string) {
	registrationName := fmt.Sprintf("Register%sProvider", service.Name)
	
	g.writeLine(fmt.Sprintf("// %s registers the %s provider with the DI container", registrationName, service.Name))
	g.writeLine(fmt.Sprintf("func %s(container *core.DIContainer) error {", registrationName))
	g.indent()
	
	// Register the provider with the specified scope
	serviceToken := strings.ToLower(service.Name)
	
	switch scope {
	case "singleton":
		g.writeLine(fmt.Sprintf("return container.RegisterSingleton(\"%s\", %s)", serviceToken, factoryName))
	case "transient":
		g.writeLine(fmt.Sprintf("return container.RegisterTransient(\"%s\", %s)", serviceToken, factoryName))
	case "request", "scoped":
		g.writeLine(fmt.Sprintf("return container.RegisterScoped(\"%s\", %s)", serviceToken, factoryName))
	default:
		// Default to singleton
		g.writeLine(fmt.Sprintf("return container.RegisterSingleton(\"%s\", %s)", serviceToken, factoryName))
	}
	
	g.unindent()
	g.writeLine("}")
}

// generateInjectionTag generates injection struct tags
func (g *CodeGenerator) generateInjectionTag(field *FieldNode) string {
	if field.Tag != "" {
		return field.Tag
	}

	// Check for @Inject() decorator
	injectDecorator := g.getInjectDecorator(field)
	if injectDecorator != nil {
		return g.generateInjectTag(injectDecorator, field)
	}

	// Generate inject tag based on field name/type if no decorator is present
	return `inject:""`
}

// getInjectDecorator finds @Inject() decorator on a field
func (g *CodeGenerator) getInjectDecorator(field *FieldNode) *DecoratorNode {
	for _, decorator := range field.Decorators {
		if decorator.Name == "Inject" {
			return decorator
		}
	}
	return nil
}

// generateInjectTag generates inject struct tag from @Inject() decorator
func (g *CodeGenerator) generateInjectTag(decorator *DecoratorNode, field *FieldNode) string {
	// Default inject tag
	tag := `inject:""`
	
	// If no arguments, use field name as token
	if len(decorator.Args) == 0 {
		// Convert field name to lowercase for token
		token := strings.ToLower(field.Name)
		tag = fmt.Sprintf(`inject:"%s"`, token)
		return tag
	}
	
	// Process decorator arguments
	for _, arg := range decorator.Args {
		// Handle string argument (injection token)
		if tokenValue, ok := arg.Value.(string); ok {
			tag = fmt.Sprintf(`inject:"%s"`, tokenValue)
			break
		}
		
		// Handle object argument with injection configuration
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			var parts []string
			
			// Extract token
			if token, exists := objValue["token"]; exists {
				if tokenStr, ok := token.(string); ok {
					parts = append(parts, fmt.Sprintf(`inject:"%s"`, tokenStr))
				}
			} else {
				// Use field name as default token
				token := strings.ToLower(field.Name)
				parts = append(parts, fmt.Sprintf(`inject:"%s"`, token))
			}
			
			// Extract optional (for optional injection)
			if optional, exists := objValue["optional"]; exists {
				if optBool, ok := optional.(bool); ok && optBool {
					parts = append(parts, `optional:"true"`)
				}
			}
			
			// Extract scope
			if scope, exists := objValue["scope"]; exists {
				if scopeStr, ok := scope.(string); ok {
					parts = append(parts, fmt.Sprintf(`scope:"%s"`, scopeStr))
				}
			}
			
			if len(parts) > 0 {
				tag = strings.Join(parts, " ")
			}
			break
		}
	}
	
	return tag
}

// getInjectableScope extracts the scope from @Scope or @Injectable decorators
func (g *CodeGenerator) getInjectableScope(service *ServiceDeclaration) string {
	// First, check for dedicated @Scope() decorator - it takes precedence
	scopeDecorator := g.getDecorator(service.Decorators, "Scope")
	if scopeDecorator != nil {
		scope := g.getScopeFromDecorator(scopeDecorator)
		if scope != "" {
			return scope
		}
	}
	
	// Fall back to @Injectable decorator scope
	injectableDecorator := g.getDecorator(service.Decorators, "Injectable")
	if injectableDecorator != nil {
		scope := g.getScopeFromDecorator(injectableDecorator)
		if scope != "" {
			return scope
		}
	}
	
	return "singleton" // Default scope
}

// getScopeFromDecorator extracts scope value from any decorator (Injectable or Scope)
func (g *CodeGenerator) getScopeFromDecorator(decorator *DecoratorNode) string {
	// Check for scope in decorator arguments
	for _, arg := range decorator.Args {
		// Handle string argument (scope)
		if scopeValue, ok := arg.Value.(string); ok {
			return g.normalizeScopeName(scopeValue)
		}
		
		// Handle object argument with scope property
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			if scope, exists := objValue["scope"]; exists {
				if scopeStr, ok := scope.(string); ok {
					return g.normalizeScopeName(scopeStr)
				}
			}
		}
	}
	
	return ""
}

// normalizeScopeName normalizes scope names to standard values
func (g *CodeGenerator) normalizeScopeName(scope string) string {
	switch strings.ToLower(scope) {
	case "singleton", "single":
		return "singleton"
	case "transient", "prototype", "instance":
		return "transient"
	case "request", "scoped", "req":
		return "request"
	default:
		return scope // Return as-is for custom scopes
	}
}

// generateFieldDependencyInjection generates dependency injection code for a field
func (g *CodeGenerator) generateFieldDependencyInjection(field *FieldNode) {
	// Get injection configuration from @Inject decorator or default
	injectionConfig := g.getFieldInjectionConfig(field)
	
	g.writeLine(fmt.Sprintf("// Inject %s", field.Name))
	
	if injectionConfig.Optional {
		// Optional dependency - don't fail if not found
		g.writeLine(fmt.Sprintf("if dep, exists := container.GetOptional(\"%s\"); exists {", injectionConfig.Token))
		g.indent()
		g.writeLine(fmt.Sprintf("if typedDep, ok := dep.(%s); ok {", field.Type))
		g.indent()
		g.writeLine(fmt.Sprintf("instance.%s = typedDep", field.Name))
		g.unindent()
		g.writeLine("}")
		g.unindent()
		g.writeLine("}")
	} else {
		// Required dependency
		g.writeLine(fmt.Sprintf("dep, err := container.Get(\"%s\")", injectionConfig.Token))
		g.writeLine("if err != nil {")
		g.indent()
		g.writeLine(fmt.Sprintf("return nil, fmt.Errorf(\"failed to inject %s: %%w\", err)", field.Name))
		g.unindent()
		g.writeLine("}")
		g.writeLine(fmt.Sprintf("if typedDep, ok := dep.(%s); ok {", field.Type))
		g.indent()
		g.writeLine(fmt.Sprintf("instance.%s = typedDep", field.Name))
		g.unindent()
		g.writeLine("} else {")
		g.indent()
		g.writeLine(fmt.Sprintf("return nil, fmt.Errorf(\"dependency %s has wrong type, expected %s\")", injectionConfig.Token, field.Type))
		g.unindent()
		g.writeLine("}")
	}
	
	g.writeLine("")
}

// getFieldInjectionConfig extracts injection configuration from field
func (g *CodeGenerator) getFieldInjectionConfig(field *FieldNode) FieldInjectionConfig {
	config := FieldInjectionConfig{
		Token:    strings.ToLower(field.Name), // Default to field name
		Optional: false,
		Scope:    "singleton",
	}
	
	// Check for @Inject decorator
	injectDecorator := g.getInjectDecorator(field)
	if injectDecorator == nil {
		return config
	}
	
	// Process decorator arguments
	for _, arg := range injectDecorator.Args {
		// Handle string argument (injection token)
		if tokenValue, ok := arg.Value.(string); ok {
			config.Token = tokenValue
			break
		}
		
		// Handle object argument with injection configuration
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			// Extract token
			if token, exists := objValue["token"]; exists {
				if tokenStr, ok := token.(string); ok {
					config.Token = tokenStr
				}
			}
			
			// Extract optional
			if optional, exists := objValue["optional"]; exists {
				if optBool, ok := optional.(bool); ok {
					config.Optional = optBool
				}
			}
			
			// Extract scope
			if scope, exists := objValue["scope"]; exists {
				if scopeStr, ok := scope.(string); ok {
					config.Scope = scopeStr
				}
			}
			break
		}
	}
	
	return config
}