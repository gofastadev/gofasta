package codegen

import (
	"fmt"
	"strings"
)

// reset resets the code generator state
func (g *CodeGenerator) reset() {
	g.generatedCode.Reset()
	g.indentLevel = 0
	g.imports = []string{}
	g.webSocketFunctions = []*WebSocketFunctionDeclaration{}
}

// writeLine writes a line with proper indentation
func (g *CodeGenerator) writeLine(line string) {
	for i := 0; i < g.indentLevel; i++ {
		g.generatedCode.WriteString("\t")
	}
	g.generatedCode.WriteString(line)
	g.generatedCode.WriteString("\n")
}

// indent increases indentation level
func (g *CodeGenerator) indent() {
	g.indentLevel++
}

// unindent decreases indentation level
func (g *CodeGenerator) unindent() {
	if g.indentLevel > 0 {
		g.indentLevel--
	}
}

// hasDecorator checks if a decorator exists in a list
func (g *CodeGenerator) hasDecorator(decorators []*DecoratorNode, name string) bool {
	for _, decorator := range decorators {
		if decorator.Name == name {
			return true
		}
	}
	return false
}

// getDecorator finds a decorator by name
func (g *CodeGenerator) getDecorator(decorators []*DecoratorNode, name string) *DecoratorNode {
	for _, decorator := range decorators {
		if decorator.Name == name {
			return decorator
		}
	}
	return nil
}

// getDecoratorArgValue gets decorator argument value
func (g *CodeGenerator) getDecoratorArgValue(decorator *DecoratorNode, index int) string {
	if index < len(decorator.Args) {
		if val, ok := decorator.Args[index].Value.(string); ok {
			return val
		}
	}
	return ""
}

// findDecorator finds a decorator by name (alternative implementation)
func (g *CodeGenerator) findDecorator(decorators []*DecoratorNode, name string) *DecoratorNode {
	for _, decorator := range decorators {
		if decorator.Name == name {
			return decorator
		}
	}
	return nil
}

// getParameterDecorators gets decorators for a parameter
func (g *CodeGenerator) getParameterDecorators(param *ParameterNode) []*DecoratorNode {
	return param.Decorators
}

// collectImports collects all necessary imports for a file
func (g *CodeGenerator) collectImports(file *GofaFile) {
	// Standard imports for Gofasta
	g.addImport("github.com/healtronlabs/gofasta/packages/core")
	g.addImport("github.com/healtronlabs/gofasta/packages/http")
	
	// Standard Go library imports for query parameter handling
	g.addImport("strconv")
	g.addImport("strings")
	g.addImport("net/http")
	g.addImport("fmt")
	g.addImport("time")

	// Check if we need additional imports based on decorators
	for _, decl := range file.Declarations {
		g.collectImportsFromDeclaration(decl)
	}
}

// collectImportsFromDeclaration collects imports from a declaration
func (g *CodeGenerator) collectImportsFromDeclaration(decl GofaDeclaration) {
	switch d := decl.(type) {
	case *ControllerDeclaration:
		// Controllers need HTTP package
		g.addImport("github.com/healtronlabs/gofasta/packages/http")

	case *ServiceDeclaration:
		// Check for database-related imports
		for _, field := range d.Fields {
			if strings.Contains(field.Type, "Repository") {
				g.addImport("github.com/healtronlabs/gofasta/packages/orm")
			}
		}

	case *ModuleDeclaration:
		// Modules need core package
		g.addImport("github.com/healtronlabs/gofasta/packages/core")
		
	case *TestSuiteDeclaration:
		// Add testing imports
		g.addImport("testing")
		g.addImport("github.com/stretchr/testify/assert")
		g.addImport("github.com/stretchr/testify/suite")
	case *FactoryDeclaration:
		// Add factory imports
		g.addImport("math/rand")
		g.addImport("time")
		g.addImport("fmt")
	case *MockDeclaration:
		// Add mock imports
		g.addImport("testing")
		g.addImport("errors")
		g.addImport("fmt")
	case *TestModuleDeclaration:
		// Add test module imports
		g.addImport("testing")
		g.addImport("github.com/healtronlabs/gofasta/packages/core")
	}
}

// addImport adds an import if not already present
func (g *CodeGenerator) addImport(imp string) {
	for _, existing := range g.imports {
		if existing == imp {
			return
		}
	}
	g.imports = append(g.imports, imp)
}

// generateMethodSignature generates method signature for controllers or services
func (g *CodeGenerator) generateMethodSignature(receiverType string, method *MethodNode, isController bool) string {
	var sig strings.Builder

	// Receiver
	var receiverChar string
	if isController {
		receiverChar = "c" // Standard 'c' for controllers
	} else {
		receiverChar = "s" // Standard 's' for services
	}
	sig.WriteString(fmt.Sprintf("func (%s *%s) %s", receiverChar, receiverType, method.Name))

	// Parameters
	sig.WriteString("(")
	if isController {
		// Controllers always get HTTP context as first parameter
		sig.WriteString("ctx *httpPackage.RequestContext")
		
		// For controllers, only include non-decorated parameters in the signature
		// Decorated parameters are extracted in the method body
		nonDecoratedParams := []string{}
		for _, param := range method.Params {
			if len(param.Decorators) == 0 {
				nonDecoratedParams = append(nonDecoratedParams, fmt.Sprintf("%s %s", param.Name, param.Type))
			}
		}
		if len(nonDecoratedParams) > 0 {
			sig.WriteString(", ")
			sig.WriteString(strings.Join(nonDecoratedParams, ", "))
		}
	} else {
		// For services, include all parameters
		for i, param := range method.Params {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("%s %s", param.Name, param.Type))
		}
	}
	sig.WriteString(")")

	// Return type
	if method.ReturnType != "" && method.ReturnType != "void" {
		if isController {
			// Controllers don't return values, they write to context
		} else {
			sig.WriteString(" " + method.ReturnType)
		}
	}

	return sig.String()
}

// convertDecoratorArgsToInterface converts decorator args to interface slice
func (g *CodeGenerator) convertDecoratorArgsToInterface(args []DecoratorArg) []interface{} {
	var result []interface{}
	for _, arg := range args {
		result = append(result, arg.Value)
	}
	return result
}

// extractTagContent extracts content between quotes after a key in struct tags
func (g *CodeGenerator) extractTagContent(tag, key string) string {
	// Extract content between quotes after key:
	// `validate:"@IsEmail() @Min(18)"` -> "@IsEmail() @Min(18)"
	keyPattern := key + `:"` 
	startIndex := strings.Index(tag, keyPattern)
	if startIndex == -1 {
		return ""
	}
	
	startIndex += len(keyPattern)
	endIndex := strings.Index(tag[startIndex:], `"`)
	if endIndex == -1 {
		return ""
	}
	
	return tag[startIndex : startIndex+endIndex]
}

// extractDecorators extracts individual decorators from validation content
func (g *CodeGenerator) extractDecorators(content string) []string {
	var decorators []string
	content = strings.TrimSpace(content)
	
	// Split by @ but keep the @
	parts := strings.Split(content, "@")
	for i, part := range parts {
		if i == 0 && part == "" {
			continue // Skip empty first part
		}
		if part != "" {
			decorators = append(decorators, "@"+part)
		}
	}
	
	return decorators
}

// formatValue formats a value for Go code generation
func (g *CodeGenerator) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatValueList formats a list of values for Go code generation
func (g *CodeGenerator) formatValueList(values []interface{}) string {
	var formattedValues []string
	for _, value := range values {
		formattedValues = append(formattedValues, g.formatValue(value))
	}
	return strings.Join(formattedValues, ", ")
}

// parseConstraint parses a constraint string and returns a ParamConstraint
func (g *CodeGenerator) parseConstraint(constraintStr string) ParamConstraint {
	constraint := ParamConstraint{}

	// Handle constraints with values like "min(1)", "max(100)", "range(1,100)", "regex(\\d+)"
	if strings.Contains(constraintStr, "(") && strings.Contains(constraintStr, ")") {
		openParen := strings.Index(constraintStr, "(")
		closeParen := strings.LastIndex(constraintStr, ")")

		constraint.Type = constraintStr[:openParen]
		valueStr := constraintStr[openParen+1 : closeParen]

		// Handle range constraints with two values
		if constraint.Type == "range" && strings.Contains(valueStr, ",") {
			values := strings.Split(valueStr, ",")
			if len(values) == 2 {
				constraint.Value = strings.TrimSpace(values[0])
				constraint.Value2 = strings.TrimSpace(values[1])
			}
		} else {
			constraint.Value = valueStr
		}
	} else {
		// Simple constraints without values like "int", "alpha", "bool"
		constraint.Type = constraintStr
	}

	return constraint
}

