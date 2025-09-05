package parsing

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

// Parser represents the parser state
type Parser struct {
	lexer     *Lexer
	currToken Token
	peekToken Token
	errors    []string
}

// NewParser creates a new parser instance
func NewParser(lexer *Lexer) *Parser {
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}

	// Read two tokens, so currToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances both currToken and peekToken
func (p *Parser) nextToken() {
	p.currToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// expectToken checks if the current token matches the expected type and advances
func (p *Parser) expectToken(tokenType TokenType) bool {
	if p.currToken.Type == tokenType {
		p.nextToken()
		return true
	}

	p.addError(fmt.Sprintf("expected %s, got %s at line %d, column %d",
		tokenTypeNames[tokenType], tokenTypeNames[p.currToken.Type],
		p.currToken.Line, p.currToken.Column))
	return false
}

// addError adds an error to the parser
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

// Errors returns all parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// ParseFile parses a complete .gofa file
func (p *Parser) ParseFile() (*core.GofaFile, error) {
	file := &core.GofaFile{
		Position:     p.currToken.Position,
		Declarations: []core.GofaDeclaration{},
		Decorators:   []*core.DecoratorNode{},
		Comments:     []*ast.CommentGroup{},
	}

	// Skip comments at the beginning
	p.skipComments()

	// Parse package declaration
	if p.currToken.Type == PACKAGE {
		p.nextToken() // consume 'package'
		if p.currToken.Type == IDENT {
			file.Package = &ast.Ident{Name: p.currToken.Literal}
			p.nextToken()
		} else {
			p.addError("expected package name")
		}
	}

	// Parse imports
	for p.currToken.Type == IMPORT {
		importSpec := p.parseImport()
		if importSpec != nil {
			file.Imports = append(file.Imports, importSpec)
		}
	}

	// Parse file-level decorators and declarations
	for p.currToken.Type != EOF {
		p.skipComments()

		if p.currToken.Type == DECORATOR {
			// Collect all consecutive decorators
			var decorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					decorators = append(decorators, decorator)
				} else {
					// If decorator parsing failed, advance token to avoid infinite loop
					p.nextToken()
					break
				}
			}

			// After parsing all decorators, check if current token is a declaration
			if p.currToken.Type == TYPE || p.currToken.Type == FUNC {
				if p.currToken.Type == FUNC {
					// Check if this is a method for the last controller or test suite
					if len(file.Declarations) > 0 {
						if controller, ok := file.Declarations[len(file.Declarations)-1].(*core.ControllerDeclaration); ok {
							// Parse as method and attach to controller
							method := p.parseMethod()
							if method != nil {
								// Attach decorators to method
								for _, decorator := range decorators {
									method.Decorators = append(method.Decorators, decorator)
								}
								controller.Methods = append(controller.Methods, method)
							}
							continue
						} else if testSuite, ok := file.Declarations[len(file.Declarations)-1].(*core.TestSuiteDeclaration); ok {
							// Parse as method and attach to test suite
							method := p.parseMethod()
							if method != nil {
								// Attach decorators to method
								for _, decorator := range decorators {
									method.Decorators = append(method.Decorators, decorator)
								}
								testSuite.Methods = append(testSuite.Methods, method)
							}
							continue
						} else if factory, ok := file.Declarations[len(file.Declarations)-1].(*core.FactoryDeclaration); ok {
							// Parse as method and attach to factory (for @Trait methods)
							method := p.parseMethod()
							if method != nil {
								// Attach decorators to method
								for _, decorator := range decorators {
									method.Decorators = append(method.Decorators, decorator)
								}
								factory.Methods = append(factory.Methods, method)
							}
							continue
						}
					}
				}

				// Parse the declaration and attach all decorators
				decl := p.parseDeclarationWithDecorators(decorators)
				if decl != nil {
					for _, decorator := range decorators {
						p.attachDecoratorToDeclaration(decorator, decl)
					}

					// Validate WebSocket declarations after decorators are attached
					p.postProcessWebSocketValidation(decl)

					file.Declarations = append(file.Declarations, decl)
				}
			} else {
				// File-level decorators
				if len(decorators) > 10 { // Arbitrary threshold for suspicious input
					p.addError(fmt.Sprintf("too many consecutive decorators (%d) without a declaration", len(decorators)))
				} else {
					for _, decorator := range decorators {
						file.Decorators = append(file.Decorators, decorator)
					}
				}
			}
		} else if p.currToken.Type == TYPE || p.currToken.Type == FUNC {
			decl := p.parseDeclaration()
			if decl != nil {
				file.Declarations = append(file.Declarations, decl)
			}
		} else {
			// Unknown token, add error and advance to avoid infinite loop
			if p.currToken.Type == ILLEGAL {
				p.addError(fmt.Sprintf("illegal token '%s' at line %d", p.currToken.Literal, p.currToken.Line))
			} else {
				p.addError(fmt.Sprintf("unexpected token %s at line %d", tokenTypeNames[p.currToken.Type], p.currToken.Line))
			}
			p.nextToken()
		}
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parsing errors: %s", strings.Join(p.errors, "; "))
	}

	// Validate that the file has meaningful content
	if file.Package == nil && len(file.Declarations) == 0 && len(file.Imports) == 0 {
		return nil, fmt.Errorf("empty .gofa file: file must contain at least a package declaration, imports, or declarations")
	}

	return file, nil
}

// parseImport parses an import declaration
func (p *Parser) parseImport() *ast.ImportSpec {
	if !p.expectToken(IMPORT) {
		return nil
	}

	importSpec := &ast.ImportSpec{}

	if p.currToken.Type == STRING {
		importSpec.Path = &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + p.currToken.Literal + `"`,
		}
		p.nextToken()
	} else {
		p.addError("expected import path")
		return nil
	}

	return importSpec
}

// parseDeclaration parses a top-level declaration
func (p *Parser) parseDeclaration() core.GofaDeclaration {
	return p.parseDeclarationWithDecorators(nil)
}

// parseDeclarationWithDecorators parses a declaration with access to decorators
func (p *Parser) parseDeclarationWithDecorators(decorators []*core.DecoratorNode) core.GofaDeclaration {
	switch p.currToken.Type {
	case TYPE:
		return p.parseTypeDeclarationWithDecorators(decorators)
	case FUNC:
		if p.hasWebSocketFunctionDecorator(decorators) {
			return p.parseWebSocketFunctionDeclaration(decorators)
		}
		return p.parseFunctionDeclaration()
	case EOF:
		return nil
	default:
		p.addError(fmt.Sprintf("unexpected token %s at line %d", tokenTypeNames[p.currToken.Type], p.currToken.Line))
		// Skip to next known token to avoid infinite loop
		p.skipToNextDeclaration()
		return nil
	}
}

// parseTypeDeclarationWithDecorators parses a type declaration with access to decorators
func (p *Parser) parseTypeDeclarationWithDecorators(decorators []*core.DecoratorNode) core.GofaDeclaration {
	if !p.expectToken(TYPE) {
		return nil
	}

	if p.currToken.Type != IDENT {
		p.addError("expected type name")
		return nil
	}

	typeName := p.currToken.Literal
	p.nextToken()

	if !p.expectToken(STRUCT) {
		return nil
	}

	if !p.expectToken(LBRACE) {
		return nil
	}

	// Check for explicit decorator types first
	if p.hasWebSocketGatewayDecorator(decorators) {
		return p.parseWebSocketGatewayDeclaration(typeName, decorators)
	}

	if p.hasTestSuiteDecorator(decorators) {
		return p.parseTestSuiteDeclaration(typeName)
	}

	if p.hasFactoryDecorator(decorators) {
		return p.parseFactoryDeclaration(typeName, decorators)
	}

	if p.hasMockDecorator(decorators) {
		return p.parseMockDeclaration(typeName, decorators)
	}

	if p.hasTestModuleDecorator(decorators) {
		return p.parseTestModuleDeclaration(typeName, decorators)
	}

	// Determine declaration type based on naming convention
	if strings.HasSuffix(typeName, "Controller") {
		return p.parseControllerDeclaration(typeName)
	} else if strings.HasSuffix(typeName, "Service") {
		return p.parseServiceDeclaration(typeName)
	} else if strings.HasSuffix(typeName, "Module") {
		return p.parseModuleDeclaration(typeName)
	} else if strings.HasSuffix(typeName, "Tests") || strings.HasSuffix(typeName, "TestSuite") {
		return p.parseTestSuiteDeclaration(typeName)
	}

	// Default to service declaration
	return p.parseServiceDeclaration(typeName)
}

// Helper functions for decorator checking
func (p *Parser) hasTestSuiteDecorator(decorators []*core.DecoratorNode) bool {
	if decorators == nil {
		return false
	}
	for _, decorator := range decorators {
		if decorator.Name == "TestSuite" {
			return true
		}
	}
	return false
}

func (p *Parser) hasFactoryDecorator(decorators []*core.DecoratorNode) bool {
	if decorators == nil {
		return false
	}
	for _, decorator := range decorators {
		if decorator.Name == "Factory" {
			return true
		}
	}
	return false
}

func (p *Parser) hasMockDecorator(decorators []*core.DecoratorNode) bool {
	if decorators == nil {
		return false
	}
	for _, decorator := range decorators {
		if decorator.Name == "Mock" {
			return true
		}
	}
	return false
}

func (p *Parser) hasTestModuleDecorator(decorators []*core.DecoratorNode) bool {
	if decorators == nil {
		return false
	}
	for _, decorator := range decorators {
		if decorator.Name == "TestModule" {
			return true
		}
	}
	return false
}

// hasWebSocketGatewayDecorator checks if decorators contain a WebSocketGateway decorator
func (p *Parser) hasWebSocketGatewayDecorator(decorators []*core.DecoratorNode) bool {
	if decorators == nil {
		return false
	}
	for _, decorator := range decorators {
		if decorator.Name == "WebSocketGateway" {
			return true
		}
	}
	return false
}

// hasWebSocketFunctionDecorator checks if any decorator is a WebSocket function decorator
func (p *Parser) hasWebSocketFunctionDecorator(decorators []*core.DecoratorNode) bool {
	if decorators == nil {
		return false
	}
	for _, decorator := range decorators {
		switch decorator.Name {
		case "SubscribeMessage", "OnMessage", "MessagePattern",
			"OnGatewayConnection", "OnGatewayDisconnect", "OnGatewayInit":
			return true
		}
	}
	return false
}

// parseControllerDeclaration parses a controller declaration
func (p *Parser) parseControllerDeclaration(name string) *core.ControllerDeclaration {
	controller := &core.ControllerDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{},
		Fields:     []*core.FieldNode{},
		Methods:    []*core.MethodNode{},
	}

	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			// Parse field decorators
			var decorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					decorators = append(decorators, decorator)
				} else {
					p.nextToken()
					break
				}
			}

			// Parse the field after decorators
			if p.currToken.Type == IDENT {
				field := p.parseField()
				if field != nil {
					field.Decorators = decorators
					controller.Fields = append(controller.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				controller.Fields = append(controller.Fields, field)
			} else {
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	// Parse standalone functions following the struct as methods
	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			controller.Methods = append(controller.Methods, method)
		} else {
			break
		}
	}

	return controller
}

// parseServiceDeclaration parses a service declaration
func (p *Parser) parseServiceDeclaration(name string) *core.ServiceDeclaration {
	service := &core.ServiceDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{},
		Fields:     []*core.FieldNode{},
		Methods:    []*core.MethodNode{},
	}

	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			var decorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					decorators = append(decorators, decorator)
				} else {
					p.nextToken()
					break
				}
			}

			if p.currToken.Type == IDENT {
				field := p.parseField()
				if field != nil {
					field.Decorators = decorators
					service.Fields = append(service.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				service.Fields = append(service.Fields, field)
			} else {
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	// Parse standalone functions following the struct as methods
	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			service.Methods = append(service.Methods, method)
		} else {
			break
		}
	}

	return service
}

// parseModuleDeclaration parses a module declaration
func (p *Parser) parseModuleDeclaration(name string) *core.ModuleDeclaration {
	module := &core.ModuleDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{},
	}

	// Skip to closing brace
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		p.nextToken()
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	return module
}

// parseTestSuiteDeclaration parses a test suite declaration
func (p *Parser) parseTestSuiteDeclaration(name string) *core.TestSuiteDeclaration {
	testSuite := &core.TestSuiteDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{},
		Fields:     []*core.FieldNode{},
		Methods:    []*core.MethodNode{},
	}

	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			var decorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					decorators = append(decorators, decorator)
				} else {
					p.nextToken()
					break
				}
			}

			if p.currToken.Type == IDENT {
				field := p.parseField()
				if field != nil {
					field.Decorators = decorators
					testSuite.Fields = append(testSuite.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				testSuite.Fields = append(testSuite.Fields, field)
			} else {
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	// Parse standalone functions following the struct as methods
	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			testSuite.Methods = append(testSuite.Methods, method)
		} else {
			break
		}
	}

	return testSuite
}

// parseFactoryDeclaration parses a factory declaration
func (p *Parser) parseFactoryDeclaration(name string, decorators []*core.DecoratorNode) *core.FactoryDeclaration {
	targetType := name
	if strings.HasSuffix(name, "Factory") {
		targetType = strings.TrimSuffix(name, "Factory")
	}

	factory := &core.FactoryDeclaration{
		Name:       name,
		TargetType: targetType,
		Position:   p.currToken.Position,
		Decorators: decorators,
		Fields:     []*core.FieldNode{},
		Methods:    []*core.MethodNode{},
	}

	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			var fieldDecorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					fieldDecorators = append(fieldDecorators, decorator)
				} else {
					p.nextToken()
					break
				}
			}

			if p.currToken.Type == IDENT {
				field := p.parseField()
				if field != nil {
					field.Decorators = fieldDecorators
					factory.Fields = append(factory.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				factory.Fields = append(factory.Fields, field)
			} else {
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			factory.Methods = append(factory.Methods, method)
		}
	}

	return factory
}

// parseMockDeclaration parses a mock declaration
func (p *Parser) parseMockDeclaration(name string, decorators []*core.DecoratorNode) *core.MockDeclaration {
	targetType := name
	if strings.HasPrefix(name, "Mock") {
		targetType = strings.TrimPrefix(name, "Mock")
	}

	mock := &core.MockDeclaration{
		Name:       name,
		TargetType: targetType,
		Position:   p.currToken.Position,
		Decorators: decorators,
		Fields:     []*core.FieldNode{},
		Methods:    []*core.MethodNode{},
	}

	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			var fieldDecorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					fieldDecorators = append(fieldDecorators, decorator)
				} else {
					p.nextToken()
					break
				}
			}

			if p.currToken.Type == IDENT {
				field := p.parseField()
				if field != nil {
					field.Decorators = fieldDecorators
					mock.Fields = append(mock.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				mock.Fields = append(mock.Fields, field)
			} else {
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			mock.Methods = append(mock.Methods, method)
		}
	}

	return mock
}

// parseTestModuleDeclaration parses a test module declaration
func (p *Parser) parseTestModuleDeclaration(name string, decorators []*core.DecoratorNode) *core.TestModuleDeclaration {
	testModule := &core.TestModuleDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: decorators,
		Providers:  []string{},
		Imports:    []string{},
		Fields:     []*core.FieldNode{},
		Methods:    []*core.MethodNode{},
	}

	// Extract providers and imports from @TestModule() decorator arguments
	for _, decorator := range decorators {
		if decorator.Name == "TestModule" {
			for _, arg := range decorator.Args {
				if argMap, ok := arg.Value.(map[string]interface{}); ok {
					if providers, exists := argMap["providers"]; exists {
						if providerArray, ok := providers.([]interface{}); ok {
							for _, provider := range providerArray {
								if providerStr, ok := provider.(string); ok {
									testModule.Providers = append(testModule.Providers, providerStr)
								}
							}
						}
					}
					if imports, exists := argMap["imports"]; exists {
						if importArray, ok := imports.([]interface{}); ok {
							for _, importModule := range importArray {
								if importStr, ok := importModule.(string); ok {
									testModule.Imports = append(testModule.Imports, importStr)
								}
							}
						}
					}
				}
			}
		}
	}

	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			var fieldDecorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					fieldDecorators = append(fieldDecorators, decorator)
				}
			}

			field := p.parseField()
			if field != nil {
				field.Decorators = fieldDecorators
				testModule.Fields = append(testModule.Fields, field)
			}
		} else if p.currToken.Type == FUNC {
			method := p.parseMethod()
			if method != nil {
				testModule.Methods = append(testModule.Methods, method)
			}
		} else {
			field := p.parseField()
			if field != nil {
				testModule.Fields = append(testModule.Fields, field)
			}
		}
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			testModule.Methods = append(testModule.Methods, method)
		}
	}

	return testModule
}

// parseField parses a struct field
func (p *Parser) parseField() *core.FieldNode {
	if p.currToken.Type != IDENT {
		return nil
	}

	field := &core.FieldNode{
		Name:       p.currToken.Literal,
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{},
	}
	p.nextToken()

	// Parse type
	fieldType := p.parseType()
	field.Type = fieldType

	// Parse struct tag if present
	if p.currToken.Type == STRING {
		field.Tag = p.currToken.Literal
		p.nextToken()
	}

	return field
}

// parseMethod parses a method declaration
func (p *Parser) parseMethod() *core.MethodNode {
	if !p.expectToken(FUNC) {
		return nil
	}

	// Check if this has a receiver (method) or is a standalone function
	if p.currToken.Type == LPAREN {
		p.nextToken()

		// Skip receiver type
		for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
			p.nextToken()
		}

		if !p.expectToken(RPAREN) {
			return nil
		}
	}

	// Parse function name
	if p.currToken.Type != IDENT {
		p.addError("expected function name")
		return nil
	}

	method := &core.MethodNode{
		Name:       p.currToken.Literal,
		Position:   p.currToken.Position,
		Params:     []*core.ParameterNode{},
		Decorators: []*core.DecoratorNode{},
	}
	p.nextToken()

	// Parse parameters
	if p.currToken.Type == LPAREN {
		p.nextToken()
		for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
			param := p.parseParameter()
			if param != nil {
				method.Params = append(method.Params, param)
			}

			if p.currToken.Type == COMMA {
				p.nextToken()
			} else if p.currToken.Type != RPAREN {
				p.addError("expected ',' or ')' in parameter list")
				break
			}
		}
		if !p.expectToken(RPAREN) {
			return nil
		}
	}

	// Parse return type if present
	if p.currToken.Type != LBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == LPAREN {
			// Complex return type
			p.nextToken()
			var returnTypes []string
			for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
				if p.currToken.Type == IDENT {
					p.nextToken()
				}
				if p.currToken.Type != COMMA && p.currToken.Type != RPAREN {
					returnTypes = append(returnTypes, p.parseType())
				}
				if p.currToken.Type == COMMA {
					p.nextToken()
				}
			}
			if p.currToken.Type == RPAREN {
				p.nextToken()
			}
			if len(returnTypes) > 0 {
				method.ReturnType = returnTypes[0]
			}
		} else {
			method.ReturnType = p.parseType()
		}
	}

	// Parse method body
	if p.currToken.Type == LBRACE {
		p.parseBlockStatement()
	}

	return method
}

// parseParameter parses a method parameter
func (p *Parser) parseParameter() *core.ParameterNode {
	param := &core.ParameterNode{
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{},
	}

	// Parse decorators first
	for p.currToken.Type == DECORATOR {
		decorator := p.parseDecorator()
		if decorator != nil {
			param.Decorators = append(param.Decorators, decorator)
		} else {
			break
		}
	}

	// Parse parameter name
	if p.currToken.Type != IDENT {
		return nil
	}

	param.Name = p.currToken.Literal
	p.nextToken()

	param.Type = p.parseType()

	return param
}

// parseType parses a type expression
func (p *Parser) parseType() string {
	var typeStr strings.Builder

	// Handle multiple pointer types
	for p.currToken.Type == MULTIPLY {
		typeStr.WriteString("*")
		p.nextToken()
	}

	// Handle slice types
	for p.currToken.Type == LBRACKET {
		typeStr.WriteString("[")
		p.nextToken()
		if p.currToken.Type == RBRACKET {
			typeStr.WriteString("]")
			p.nextToken()
		}
	}

	// Handle channel types
	if p.currToken.Type == GO_CHAN {
		typeStr.WriteString("chan")
		p.nextToken()
		if p.currToken.Type == LT {
			typeStr.WriteString("<")
			p.nextToken()
			if p.currToken.Type == MINUS {
				typeStr.WriteString("-")
				p.nextToken()
			}
		}
		typeStr.WriteString(" ")
		typeStr.WriteString(p.parseType())
		return typeStr.String()
	}

	// Handle map types
	if p.currToken.Type == GO_MAP {
		typeStr.WriteString("map[")
		p.nextToken()
		if p.currToken.Type == LBRACKET {
			p.nextToken()
		}
		keyType := p.parseType()
		typeStr.WriteString(keyType)
		if p.currToken.Type == RBRACKET {
			typeStr.WriteString("]")
			p.nextToken()
		}
		valueType := p.parseType()
		typeStr.WriteString(valueType)
		return typeStr.String()
	}

	// Handle function types
	if p.currToken.Type == FUNC {
		typeStr.WriteString("func")
		p.nextToken()

		if p.currToken.Type == LPAREN {
			typeStr.WriteString("(")
			p.nextToken()
			first := true
			for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
				if !first {
					typeStr.WriteString(", ")
				}
				paramType := p.parseType()
				typeStr.WriteString(paramType)
				first = false
				if p.currToken.Type == COMMA {
					p.nextToken()
				}
			}
			if p.currToken.Type == RPAREN {
				typeStr.WriteString(")")
				p.nextToken()
			}
		}

		if p.currToken.Type != EOF && p.currToken.Type != RBRACE &&
			p.currToken.Type != COMMA && p.currToken.Type != STRING {
			typeStr.WriteString(" ")
			returnType := p.parseType()
			typeStr.WriteString(returnType)
		}

		return typeStr.String()
	}

	// Handle interface{} specifically
	if p.currToken.Type == INTERFACE {
		typeStr.WriteString("interface{}")
		p.nextToken()
		if p.currToken.Type == LBRACE {
			p.nextToken()
			if p.currToken.Type == RBRACE {
				p.nextToken()
			}
		}
		return typeStr.String()
	}

	// Handle struct types
	if p.currToken.Type == STRUCT {
		typeStr.WriteString("struct")
		p.nextToken()

		if p.currToken.Type == LBRACE {
			typeStr.WriteString(" {")
			p.nextToken()

			for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
				if p.currToken.Type == IDENT {
					typeStr.WriteString("\n\t\t")
					typeStr.WriteString(p.currToken.Literal)
					p.nextToken()

					typeStr.WriteString(" ")
					fieldType := p.parseType()
					typeStr.WriteString(fieldType)

					if p.currToken.Type == STRING {
						typeStr.WriteString(" ")
						typeStr.WriteString(p.currToken.Literal)
						p.nextToken()
					}
				} else {
					p.nextToken()
				}
			}

			if p.currToken.Type == RBRACE {
				typeStr.WriteString("\n\t}")
				p.nextToken()
			}
		}
		return typeStr.String()
	}

	// Base type
	if p.currToken.Type == IDENT || isGoType(p.currToken.Type) {
		typeStr.WriteString(p.currToken.Literal)
		p.nextToken()
	}

	return typeStr.String()
}

// parseDecorator parses a decorator
func (p *Parser) parseDecorator() *core.DecoratorNode {
	if !p.expectToken(DECORATOR) {
		return nil
	}

	if p.currToken.Type != IDENT {
		p.addError("expected decorator name")
		return nil
	}

	decorator := &core.DecoratorNode{
		Name:     p.currToken.Literal,
		Position: p.currToken.Position,
		Args:     []core.DecoratorArg{},
	}
	p.nextToken()

	// Parse decorator arguments
	if p.currToken.Type == LPAREN {
		p.nextToken()
		for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
			arg := p.parseDecoratorArg()
			if arg != nil {
				decorator.Args = append(decorator.Args, *arg)
			} else {
				p.nextToken()
			}

			if p.currToken.Type == COMMA {
				p.nextToken()
			} else if p.currToken.Type != RPAREN {
				p.addError("expected ',' or ')' in decorator arguments")
				break
			}
		}
		if !p.expectToken(RPAREN) {
			return nil
		}
	}

	return decorator
}

// parseDecoratorArg parses a decorator argument
func (p *Parser) parseDecoratorArg() *core.DecoratorArg {
	arg := &core.DecoratorArg{
		Position: p.currToken.Position,
	}

	switch p.currToken.Type {
	case STRING:
		arg.Value = p.currToken.Literal
		p.nextToken()
	case INT:
		if val, err := strconv.Atoi(p.currToken.Literal); err == nil {
			arg.Value = val
		}
		p.nextToken()
	case FLOAT:
		if val, err := strconv.ParseFloat(p.currToken.Literal, 64); err == nil {
			arg.Value = val
		}
		p.nextToken()
	case BOOLEAN:
		arg.Value = p.currToken.Literal == "true"
		p.nextToken()
	case IDENT:
		arg.Key = p.currToken.Literal
		arg.Value = p.currToken.Literal
		p.nextToken()
		if p.currToken.Type == COLON {
			p.nextToken()
			switch p.currToken.Type {
			case STRING:
				arg.Value = p.currToken.Literal
				p.nextToken()
			case INT:
				if val, err := strconv.Atoi(p.currToken.Literal); err == nil {
					arg.Value = val
				}
				p.nextToken()
			case BOOLEAN:
				arg.Value = p.currToken.Literal == "true"
				p.nextToken()
			case LBRACKET:
				arrayValues := []interface{}{}
				p.nextToken()
				for p.currToken.Type != RBRACKET && p.currToken.Type != EOF {
					switch p.currToken.Type {
					case STRING:
						arrayValues = append(arrayValues, p.currToken.Literal)
						p.nextToken()
					case INT:
						if val, err := strconv.Atoi(p.currToken.Literal); err == nil {
							arrayValues = append(arrayValues, val)
						}
						p.nextToken()
					case BOOLEAN:
						arrayValues = append(arrayValues, p.currToken.Literal == "true")
						p.nextToken()
					case IDENT:
						arrayValues = append(arrayValues, p.currToken.Literal)
						p.nextToken()
					default:
						p.nextToken()
					}
					if p.currToken.Type == COMMA {
						p.nextToken()
					}
				}
				if p.currToken.Type == RBRACKET {
					p.nextToken()
				}
				arg.Value = arrayValues
			default:
				arg.Value = p.currToken.Literal
				p.nextToken()
			}
		}
	case LBRACKET:
		arrayValues := []interface{}{}
		p.nextToken()
		for p.currToken.Type != RBRACKET && p.currToken.Type != EOF {
			switch p.currToken.Type {
			case STRING:
				arrayValues = append(arrayValues, p.currToken.Literal)
				p.nextToken()
			case INT:
				if val, err := strconv.Atoi(p.currToken.Literal); err == nil {
					arrayValues = append(arrayValues, val)
				}
				p.nextToken()
			case BOOLEAN:
				arrayValues = append(arrayValues, p.currToken.Literal == "true")
				p.nextToken()
			case IDENT:
				arrayValues = append(arrayValues, p.currToken.Literal)
				p.nextToken()
			default:
				p.nextToken()
			}
			if p.currToken.Type == COMMA {
				p.nextToken()
			}
		}
		if p.currToken.Type == RBRACKET {
			p.nextToken()
		}
		arg.Value = arrayValues
	case LBRACE:
		objectValue := make(map[string]interface{})
		p.nextToken()

		for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
			if p.currToken.Type != IDENT {
				p.nextToken()
				continue
			}
			key := p.currToken.Literal
			p.nextToken()

			if p.currToken.Type != COLON {
				p.nextToken()
				continue
			}
			p.nextToken()

			switch p.currToken.Type {
			case STRING:
				objectValue[key] = p.currToken.Literal
				p.nextToken()
			case INT:
				if val, err := strconv.Atoi(p.currToken.Literal); err == nil {
					objectValue[key] = val
				}
				p.nextToken()
			case BOOLEAN:
				objectValue[key] = p.currToken.Literal == "true"
				p.nextToken()
			case LBRACKET:
				arrayValues := []interface{}{}
				p.nextToken()
				for p.currToken.Type != RBRACKET && p.currToken.Type != EOF {
					switch p.currToken.Type {
					case STRING:
						arrayValues = append(arrayValues, p.currToken.Literal)
						p.nextToken()
					case IDENT:
						arrayValues = append(arrayValues, p.currToken.Literal)
						p.nextToken()
					default:
						p.nextToken()
					}
					if p.currToken.Type == COMMA {
						p.nextToken()
					}
				}
				if p.currToken.Type == RBRACKET {
					p.nextToken()
				}
				objectValue[key] = arrayValues
			case LBRACE:
				// Handle nested objects
				nestedObject := make(map[string]interface{})
				p.nextToken()

				for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
					if p.currToken.Type != IDENT {
						p.nextToken()
						continue
					}
					nestedKey := p.currToken.Literal
					p.nextToken()

					if p.currToken.Type != COLON {
						p.nextToken()
						continue
					}
					p.nextToken()

					// Parse nested object values
					switch p.currToken.Type {
					case STRING:
						nestedObject[nestedKey] = p.currToken.Literal
						p.nextToken()
					case INT:
						if val, err := strconv.Atoi(p.currToken.Literal); err == nil {
							nestedObject[nestedKey] = val
						}
						p.nextToken()
					case BOOLEAN:
						nestedObject[nestedKey] = p.currToken.Literal == "true"
						p.nextToken()
					default:
						nestedObject[nestedKey] = p.currToken.Literal
						p.nextToken()
					}

					if p.currToken.Type == COMMA {
						p.nextToken()
					}
				}

				if p.currToken.Type == RBRACE {
					p.nextToken()
				}
				objectValue[key] = nestedObject
			default:
				objectValue[key] = p.currToken.Literal
				p.nextToken()
			}

			if p.currToken.Type == COMMA {
				p.nextToken()
			}
		}

		if p.currToken.Type == RBRACE {
			p.nextToken()
		}
		arg.Value = objectValue
	default:
		p.addError(fmt.Sprintf("unexpected decorator argument type %d at line %d",
			int(p.currToken.Type), p.currToken.Line))
		p.nextToken()
		return nil
	}

	return arg
}

// parseFunctionDeclaration parses a standalone function
func (p *Parser) parseFunctionDeclaration() core.GofaDeclaration {
	if !p.expectToken(FUNC) {
		return nil
	}

	if p.currToken.Type == IDENT {
		p.nextToken()
	} else {
		p.addError("expected function name after 'func'")
		return nil
	}

	// Skip parameter list
	if p.currToken.Type == LPAREN {
		p.nextToken()
		for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
			p.nextToken()
		}
		if p.currToken.Type == RPAREN {
			p.nextToken()
		} else if p.currToken.Type == EOF {
			p.addError("unexpected end of file in function parameter list")
			return nil
		}
	}

	// Skip return type
	for p.currToken.Type != LBRACE && p.currToken.Type != EOF {
		p.nextToken()
	}

	// Skip function body
	if p.currToken.Type == LBRACE {
		p.parseBlockStatement()
	}

	return nil
}

// parseBlockStatement parses a block statement
func (p *Parser) parseBlockStatement() {
	if !p.expectToken(LBRACE) {
		return
	}

	depth := 1
	for depth > 0 && p.currToken.Type != EOF {
		if p.currToken.Type == LBRACE {
			depth++
		} else if p.currToken.Type == RBRACE {
			depth--
		}
		p.nextToken()
	}
}

// parseWebSocketGatewayDeclaration parses a WebSocket gateway declaration
func (p *Parser) parseWebSocketGatewayDeclaration(name string, decorators []*core.DecoratorNode) *core.WebSocketGatewayDeclaration {
	gateway := &core.WebSocketGatewayDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{}, // Empty - will be populated by attachDecoratorToDeclaration
		Fields:     []*core.FieldNode{},
		Methods:    []*core.MethodNode{},
		Config:     make(map[string]interface{}),
	}

	// Extract configuration from WebSocketGateway decorator
	for _, decorator := range decorators {
		if decorator.Name == "WebSocketGateway" {
			// Parse decorator arguments to extract port, namespace, and other config
			if len(decorator.Args) > 0 {
				// Handle simple port argument: @WebSocketGateway(8080)
				if len(decorator.Args) == 1 && decorator.Args[0].Key == "" {
					if port, ok := decorator.Args[0].Value.(int); ok {
						gateway.Port = &port
					} else if port, ok := decorator.Args[0].Value.(int64); ok {
						portInt := int(port)
						gateway.Port = &portInt
					} else if portStr, ok := decorator.Args[0].Value.(string); ok {
						if port, err := strconv.Atoi(portStr); err == nil {
							gateway.Port = &port
						}
					} else if configMap, ok := decorator.Args[0].Value.(map[string]interface{}); ok {
						// Handle object configuration: @WebSocketGateway({port: 8080, namespace: "/chat"})
						for key, value := range configMap {
							switch key {
							case "port":
								if port, ok := value.(int); ok {
									gateway.Port = &port
								} else if port, ok := value.(int64); ok {
									portInt := int(port)
									gateway.Port = &portInt
								} else if portStr, ok := value.(string); ok {
									if port, err := strconv.Atoi(portStr); err == nil {
										gateway.Port = &port
									}
								}
							case "namespace":
								if namespace, ok := value.(string); ok {
									gateway.Namespace = &namespace
								}
							default:
								// Store other configuration options
								gateway.Config[key] = value
							}
						}
					}
				} else {
					// Handle multiple separate arguments (less likely case)
					for _, arg := range decorator.Args {
						switch arg.Key {
						case "port":
							if port, ok := arg.Value.(int); ok {
								gateway.Port = &port
							} else if port, ok := arg.Value.(int64); ok {
								portInt := int(port)
								gateway.Port = &portInt
							} else if portStr, ok := arg.Value.(string); ok {
								if port, err := strconv.Atoi(portStr); err == nil {
									gateway.Port = &port
								}
							}
						case "namespace":
							if namespace, ok := arg.Value.(string); ok {
								gateway.Namespace = &namespace
							}
						default:
							// Store other configuration options
							gateway.Config[arg.Key] = arg.Value
						}
					}
				}
			}
		}
	}

	// Parse fields and methods (same as other declarations)
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			var decorators []*core.DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					decorators = append(decorators, decorator)
				} else {
					p.nextToken()
					break
				}
			}
			// Check if this is a method (starts with func) or field (ident)
			if p.currToken.Type == FUNC {
				// Method with decorators (like @SubscribeMessage)
				method := p.parseMethod()
				if method != nil {
					for _, decorator := range decorators {
						method.Decorators = append(method.Decorators, decorator)
					}
					gateway.Methods = append(gateway.Methods, method)
				}
				continue
			} else if p.currToken.Type == IDENT {
				// Field with decorators
				field := p.parseField()
				if field != nil {
					field.Decorators = decorators
					gateway.Fields = append(gateway.Fields, field)
				}
			}
		} else if p.currToken.Type == FUNC {
			// Method without decorators
			method := p.parseMethod()
			if method != nil {
				gateway.Methods = append(gateway.Methods, method)
			}
		} else if p.currToken.Type == IDENT {
			// Field without decorators
			field := p.parseField()
			if field != nil {
				gateway.Fields = append(gateway.Fields, field)
			}
		} else {
			p.nextToken()
		}
	}

	if !p.expectToken(RBRACE) {
		return nil
	}

	return gateway
}

// parseWebSocketFunctionDeclaration parses a standalone WebSocket function (OnGatewayConnection, etc.)
func (p *Parser) parseWebSocketFunctionDeclaration(decorators []*core.DecoratorNode) *core.WebSocketFunctionDeclaration {
	if !p.expectToken(FUNC) {
		return nil
	}

	if p.currToken.Type != IDENT {
		p.addError("expected function name after 'func'")
		return nil
	}

	wsFunc := &core.WebSocketFunctionDeclaration{
		Name:       p.currToken.Literal,
		Position:   p.currToken.Position,
		Decorators: []*core.DecoratorNode{},
		Params:     []*core.ParameterNode{},
	}
	p.nextToken()

	// Add decorators to the function
	for _, decorator := range decorators {
		wsFunc.Decorators = append(wsFunc.Decorators, decorator)
	}

	// Parse parameters
	if p.currToken.Type == LPAREN {
		p.nextToken()
		for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
			param := p.parseParameter()
			if param != nil {
				wsFunc.Params = append(wsFunc.Params, param)
			}

			if p.currToken.Type == COMMA {
				p.nextToken()
			} else if p.currToken.Type != RPAREN {
				p.addError("expected ',' or ')' in parameter list")
				break
			}
		}
		if !p.expectToken(RPAREN) {
			return nil
		}
	}

	// Parse return type if present
	if p.currToken.Type != LBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == LPAREN {
			// Complex return type
			p.nextToken()
			for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
				p.nextToken()
			}
			if p.currToken.Type == RPAREN {
				p.nextToken()
			}
		} else {
			wsFunc.ReturnType = p.parseType()
		}
	}

	// Parse function body
	if p.currToken.Type == LBRACE {
		p.parseBlockStatement()
	}

	return wsFunc
}

// validateWebSocketFunction validates WebSocket function decorators and signatures
func (p *Parser) validateWebSocketFunction(wsFunc *core.WebSocketFunctionDeclaration) {
	if wsFunc == nil || len(wsFunc.Decorators) == 0 {
		return
	}

	for _, decorator := range wsFunc.Decorators {
		decoratorType := core.GetDecoratorType(decorator.Name)

		// Allow WebSocket-specific decorators and middleware decorators
		if core.IsWebSocketDecorator(decoratorType) {
			switch decoratorType {
			case core.OnGatewayConnectionDecorator:
				p.validateOnGatewayConnectionDecorator(decorator, wsFunc)
			case core.OnGatewayDisconnectDecorator:
				p.validateOnGatewayDisconnectDecorator(decorator, wsFunc)
			case core.OnGatewayInitDecorator:
				p.validateOnGatewayInitDecorator(decorator, wsFunc)
			case core.SubscribeMessageDecorator:
				p.validateSubscribeMessageDecorator(decorator, wsFunc)
			default:
				// Allow other WebSocket decorators without validation for now
			}
		} else if p.isMiddlewareDecorator(decoratorType) {
			// Allow middleware decorators (UseGuards, UseInterceptors, UsePipes, UseFilters)
			// These are validated at the method level
		} else {
			p.addError(fmt.Sprintf("invalid decorator @%s for WebSocket function %s", decorator.Name, wsFunc.Name))
		}
	}
}

// isMiddlewareDecorator checks if a decorator type is a middleware decorator
func (p *Parser) isMiddlewareDecorator(decoratorType core.DecoratorType) bool {
	return decoratorType == core.UseGuardsDecorator ||
		decoratorType == core.UseInterceptorsDecorator ||
		decoratorType == core.UsePipesDecorator ||
		decoratorType == core.UseFiltersDecorator ||
		decoratorType == core.UseMiddlewareDecorator ||
		core.IsErrorHandlingDecorator(decoratorType)
}

// validateOnGatewayConnectionDecorator validates @OnGatewayConnection() decorator
func (p *Parser) validateOnGatewayConnectionDecorator(decorator *core.DecoratorNode, wsFunc *core.WebSocketFunctionDeclaration) {
	// @OnGatewayConnection() should have no arguments
	if len(decorator.Args) > 0 {
		p.addError("@OnGatewayConnection() decorator should not have arguments")
	}

	// Validate function signature - should accept connection-related parameters
	validParamTypes := map[string]bool{
		"*WebSocketClient":  true,
		"map[string]string": true, // headers
		"string":            true, // ip, client id, etc.
		"*RequestContext":   true,
		"*Session":          true,
	}

	for _, param := range wsFunc.Params {
		if !p.isValidWebSocketParamType(param.Type, validParamTypes) {
			p.addError(fmt.Sprintf("invalid parameter type '%s' for @OnGatewayConnection function '%s'. Expected: *WebSocketClient, map[string]string, string, *RequestContext, *Session, or custom data types", param.Type, wsFunc.Name))
		}
	}

	// Return type validation
	if wsFunc.ReturnType != "" && wsFunc.ReturnType != "error" {
		p.addError(fmt.Sprintf("@OnGatewayConnection function '%s' should return void or error, got '%s'", wsFunc.Name, wsFunc.ReturnType))
	}
}

// validateOnGatewayDisconnectDecorator validates @OnGatewayDisconnect() decorator
func (p *Parser) validateOnGatewayDisconnectDecorator(decorator *core.DecoratorNode, wsFunc *core.WebSocketFunctionDeclaration) {
	// @OnGatewayDisconnect() should have no arguments
	if len(decorator.Args) > 0 {
		p.addError("@OnGatewayDisconnect() decorator should not have arguments")
	}

	// Validate function signature - should accept disconnection-related parameters
	validParamTypes := map[string]bool{
		"*WebSocketClient": true,
		"string":           true, // reason, client id, etc.
		"[]string":         true, // rooms
		"*RequestContext":  true,
	}

	for _, param := range wsFunc.Params {
		if !p.isValidWebSocketParamType(param.Type, validParamTypes) {
			p.addError(fmt.Sprintf("invalid parameter type '%s' for @OnGatewayDisconnect function '%s'. Expected: *WebSocketClient, string, []string, *RequestContext, or custom data types", param.Type, wsFunc.Name))
		}
	}

	// Return type validation
	if wsFunc.ReturnType != "" && wsFunc.ReturnType != "error" {
		p.addError(fmt.Sprintf("@OnGatewayDisconnect function '%s' should return void or error, got '%s'", wsFunc.Name, wsFunc.ReturnType))
	}
}

// validateOnGatewayInitDecorator validates @OnGatewayInit() decorator
func (p *Parser) validateOnGatewayInitDecorator(decorator *core.DecoratorNode, wsFunc *core.WebSocketFunctionDeclaration) {
	// @OnGatewayInit() should have no arguments
	if len(decorator.Args) > 0 {
		p.addError("@OnGatewayInit() decorator should not have arguments")
	}

	// @OnGatewayInit functions should typically have no parameters or minimal initialization parameters
	if len(wsFunc.Params) > 2 {
		p.addError(fmt.Sprintf("@OnGatewayInit function '%s' should have minimal parameters, got %d parameters", wsFunc.Name, len(wsFunc.Params)))
	}

	// Return type validation
	if wsFunc.ReturnType != "" && wsFunc.ReturnType != "error" {
		p.addError(fmt.Sprintf("@OnGatewayInit function '%s' should return void or error, got '%s'", wsFunc.Name, wsFunc.ReturnType))
	}
}

// validateSubscribeMessageDecorator validates @SubscribeMessage() decorator
func (p *Parser) validateSubscribeMessageDecorator(decorator *core.DecoratorNode, wsFunc *core.WebSocketFunctionDeclaration) {
	// @SubscribeMessage() must have at least one argument
	if len(decorator.Args) == 0 {
		p.addError("@SubscribeMessage() decorator requires at least one event name argument")
		return
	}

	// Validate decorator arguments
	for i, arg := range decorator.Args {
		switch argValue := arg.Value.(type) {
		case string:
			if strings.TrimSpace(argValue) == "" {
				p.addError(fmt.Sprintf("@SubscribeMessage() event name cannot be empty (argument %d)", i+1))
			}
		case []interface{}:
			if len(argValue) == 0 {
				p.addError("@SubscribeMessage() event array cannot be empty")
			}
			for j, event := range argValue {
				if eventStr, ok := event.(string); ok {
					if strings.TrimSpace(eventStr) == "" {
						p.addError(fmt.Sprintf("@SubscribeMessage() event name cannot be empty (array item %d)", j+1))
					}
				} else {
					p.addError(fmt.Sprintf("@SubscribeMessage() event array must contain only strings (invalid item %d)", j+1))
				}
			}
		default:
			p.addError(fmt.Sprintf("@SubscribeMessage() argument must be a string or array of strings (argument %d)", i+1))
		}
	}

	// Validate function signature - should accept message-related parameters
	validParamTypes := map[string]bool{
		"*WebSocketClient":  true,
		"*AckCallback":      true,
		"string":            true, // event name, raw message, etc.
		"interface{}":       true, // generic data
		"map[string]string": true, // headers
		"*Session":          true,
		"*User":             true,
		"[]string":          true, // rooms, user lists, etc.
		"[]interface{}":     true, // generic arrays
		"int":               true, // room count, user count, etc.
		"bool":              true, // flags
		"float64":           true, // numeric data
	}

	hasMessageData := false
	for _, param := range wsFunc.Params {
		if !p.isValidWebSocketParamTypeWithDecorators(param, validParamTypes) {
			p.addError(fmt.Sprintf("invalid parameter type '%s' for @SubscribeMessage function '%s'. Expected: *WebSocketClient, *AckCallback, string, interface{}, map[string]string, *Session, *User, slices, or custom data types", param.Type, wsFunc.Name))
		}

		// Check for message data parameter (usually custom structs starting with *)
		if strings.HasPrefix(param.Type, "*") && param.Type != "*WebSocketClient" && param.Type != "*AckCallback" && param.Type != "*Session" && param.Type != "*User" {
			hasMessageData = true
		}
	}

	// Recommend having a message data parameter
	if !hasMessageData && len(wsFunc.Params) > 0 {
		// This is a warning, not an error - some message handlers might not need data
	}

	// Return type validation
	if wsFunc.ReturnType != "" && wsFunc.ReturnType != "error" {
		p.addError(fmt.Sprintf("@SubscribeMessage function '%s' should return void or error, got '%s'", wsFunc.Name, wsFunc.ReturnType))
	}
}

// validateWebSocketGateway validates WebSocket gateway structure and decorators
func (p *Parser) validateWebSocketGateway(gateway *core.WebSocketGatewayDeclaration) {
	if gateway == nil {
		return
	}

	// Validate @WebSocketGateway decorator
	hasGatewayDecorator := false
	for _, decorator := range gateway.Decorators {
		decoratorType := core.GetDecoratorType(decorator.Name)
		if decoratorType == core.WebSocketGatewayDecorator {
			hasGatewayDecorator = true
			p.validateWebSocketGatewayDecorator(decorator, gateway)
		}
	}

	if !hasGatewayDecorator {
		p.addError(fmt.Sprintf("WebSocket gateway '%s' must have @WebSocketGateway decorator", gateway.Name))
	}

	// Validate gateway methods
	for _, method := range gateway.Methods {
		p.validateWebSocketGatewayMethod(method, gateway)
	}

	// Validate gateway fields (dependency injection)
	for _, field := range gateway.Fields {
		p.validateWebSocketGatewayField(field, gateway)
	}
}

// validateWebSocketGatewayDecorator validates @WebSocketGateway() decorator
func (p *Parser) validateWebSocketGatewayDecorator(decorator *core.DecoratorNode, gateway *core.WebSocketGatewayDeclaration) {
	// @WebSocketGateway can have configuration arguments
	for i, arg := range decorator.Args {
		switch argValue := arg.Value.(type) {
		case int, int64:
			// Port number - validate range
			if portVal, ok := argValue.(int); ok {
				if portVal < 1 || portVal > 65535 {
					p.addError(fmt.Sprintf("@WebSocketGateway port %d is invalid, must be between 1-65535 (argument %d)", portVal, i+1))
				}
			}
		case string:
			// Namespace or configuration string
			if strings.TrimSpace(argValue) == "" {
				p.addError(fmt.Sprintf("@WebSocketGateway string argument cannot be empty (argument %d)", i+1))
			}
		case map[string]interface{}:
			// Configuration object
			p.validateWebSocketGatewayConfig(argValue, i+1)
		default:
			p.addError(fmt.Sprintf("@WebSocketGateway argument %d has invalid type, expected port number, namespace string, or configuration object", i+1))
		}
	}
}

// validateWebSocketGatewayConfig validates WebSocket gateway configuration
func (p *Parser) validateWebSocketGatewayConfig(config map[string]interface{}, argIndex int) {
	validConfigKeys := map[string]bool{
		"port":              true,
		"namespace":         true,
		"cors":              true,
		"transports":        true,
		"path":              true,
		"pingTimeout":       true,
		"pingInterval":      true,
		"maxBufferSize":     true,
		"allowEIO3":         true,
		"serveClient":       true,
		"compression":       true,
		"cookie":            true,
		"allowRequest":      true,
		"allowUpgrades":     true,
		"upgradeTimeout":    true,
		"maxHttpBufferSize": true,
		"connectTimeout":    true,
		"heartbeatTimeout":  true,
		"heartbeatInterval": true,
	}

	for key, value := range config {
		if !validConfigKeys[key] {
			p.addError(fmt.Sprintf("@WebSocketGateway config has invalid key '%s' (argument %d)", key, argIndex))
			continue
		}

		switch key {
		case "port":
			if portVal, ok := value.(int); ok {
				if portVal < 1 || portVal > 65535 {
					p.addError(fmt.Sprintf("@WebSocketGateway config port %d is invalid, must be between 1-65535 (argument %d)", portVal, argIndex))
				}
			} else {
				p.addError(fmt.Sprintf("@WebSocketGateway config 'port' must be an integer (argument %d)", argIndex))
			}
		case "namespace":
			if nsVal, ok := value.(string); ok {
				if strings.TrimSpace(nsVal) == "" {
					p.addError(fmt.Sprintf("@WebSocketGateway config 'namespace' cannot be empty (argument %d)", argIndex))
				}
			} else {
				p.addError(fmt.Sprintf("@WebSocketGateway config 'namespace' must be a string (argument %d)", argIndex))
			}
		case "cors":
			// CORS can be either a boolean (enable/disable) or an object (configuration)
			if _, isBool := value.(bool); !isBool {
				if _, isMap := value.(map[string]interface{}); !isMap {
					p.addError(fmt.Sprintf("@WebSocketGateway config 'cors' must be a boolean or configuration object (argument %d)", argIndex))
				}
			}
		case "transports":
			if transports, ok := value.([]interface{}); ok {
				validTransports := map[string]bool{"websocket": true, "polling": true}
				for _, transport := range transports {
					if transportStr, ok := transport.(string); ok {
						if !validTransports[transportStr] {
							p.addError(fmt.Sprintf("@WebSocketGateway config invalid transport '%s', must be 'websocket' or 'polling' (argument %d)", transportStr, argIndex))
						}
					} else {
						p.addError(fmt.Sprintf("@WebSocketGateway config 'transports' array must contain strings (argument %d)", argIndex))
					}
				}
			} else {
				p.addError(fmt.Sprintf("@WebSocketGateway config 'transports' must be an array (argument %d)", argIndex))
			}
		case "path":
			if pathVal, ok := value.(string); ok {
				if strings.TrimSpace(pathVal) == "" {
					p.addError(fmt.Sprintf("@WebSocketGateway config 'path' cannot be empty (argument %d)", argIndex))
				}
			} else {
				p.addError(fmt.Sprintf("@WebSocketGateway config 'path' must be a string (argument %d)", argIndex))
			}
		case "pingTimeout", "pingInterval", "maxBufferSize", "upgradeTimeout", "maxHttpBufferSize", "connectTimeout", "heartbeatTimeout", "heartbeatInterval":
			// These should be positive integers (milliseconds)
			if intVal, ok := value.(int); ok {
				if intVal <= 0 {
					p.addError(fmt.Sprintf("@WebSocketGateway config '%s' must be positive, got %d (argument %d)", key, intVal, argIndex))
				}
			} else if int64Val, ok := value.(int64); ok {
				if int64Val <= 0 {
					p.addError(fmt.Sprintf("@WebSocketGateway config '%s' must be positive, got %d (argument %d)", key, int64Val, argIndex))
				}
			} else {
				p.addError(fmt.Sprintf("@WebSocketGateway config '%s' must be an integer (argument %d)", key, argIndex))
			}
		case "allowEIO3", "serveClient", "compression", "allowUpgrades":
			// These should be boolean flags
			if _, ok := value.(bool); !ok {
				p.addError(fmt.Sprintf("@WebSocketGateway config '%s' must be a boolean (argument %d)", key, argIndex))
			}
		case "cookie", "allowRequest":
			// These can be various types (string, boolean, function, etc.) - allow any type
			// No validation needed as they're flexible configuration options
		}
	}
}

// validateWebSocketGatewayMethod validates methods in WebSocket gateway
func (p *Parser) validateWebSocketGatewayMethod(method *core.MethodNode, gateway *core.WebSocketGatewayDeclaration) {
	if method == nil {
		return
	}

	// Check for WebSocket-specific decorators on methods
	for _, decorator := range method.Decorators {
		decoratorType := core.GetDecoratorType(decorator.Name)

		if core.IsWebSocketDecorator(decoratorType) {
			switch decoratorType {
			case core.SubscribeMessageDecorator:
				// @SubscribeMessage is valid on gateway methods
				p.validateSubscribeMessageOnMethod(decorator, method, gateway)
			case core.OnGatewayConnectionDecorator, core.OnGatewayDisconnectDecorator, core.OnGatewayInitDecorator:
				p.addError(fmt.Sprintf("lifecycle decorator @%s should not be used on gateway methods, use standalone functions instead", decorator.Name))
			default:
				// Other WebSocket decorators may be valid on methods
			}
		}
	}
}

// validateSubscribeMessageOnMethod validates @SubscribeMessage on gateway methods
func (p *Parser) validateSubscribeMessageOnMethod(decorator *core.DecoratorNode, method *core.MethodNode, gateway *core.WebSocketGatewayDeclaration) {
	// Similar validation to standalone functions but adjusted for methods
	if len(decorator.Args) == 0 {
		p.addError(fmt.Sprintf("@SubscribeMessage() decorator on method '%s' requires at least one event name argument", method.Name))
		return
	}

	// Validate decorator arguments (same as standalone function)
	for i, arg := range decorator.Args {
		switch argValue := arg.Value.(type) {
		case string:
			if strings.TrimSpace(argValue) == "" {
				p.addError(fmt.Sprintf("@SubscribeMessage() event name on method '%s' cannot be empty (argument %d)", method.Name, i+1))
			}
		case []interface{}:
			if len(argValue) == 0 {
				p.addError(fmt.Sprintf("@SubscribeMessage() event array on method '%s' cannot be empty", method.Name))
			}
			for j, event := range argValue {
				if eventStr, ok := event.(string); ok {
					if strings.TrimSpace(eventStr) == "" {
						p.addError(fmt.Sprintf("@SubscribeMessage() event name on method '%s' cannot be empty (array item %d)", method.Name, j+1))
					}
				} else {
					p.addError(fmt.Sprintf("@SubscribeMessage() event array on method '%s' must contain only strings (invalid item %d)", method.Name, j+1))
				}
			}
		default:
			p.addError(fmt.Sprintf("@SubscribeMessage() argument on method '%s' must be a string or array of strings (argument %d)", method.Name, i+1))
		}
	}

	// Method signature validation
	hasReceiver := len(method.Params) > 0 && strings.Contains(method.Params[0].Type, gateway.Name)
	if hasReceiver {
		// Skip receiver parameter for validation
		params := method.Params[1:]
		p.validateSubscribeMessageParameters(params, method.Name)
	} else {
		p.validateSubscribeMessageParameters(method.Params, method.Name)
	}
}

// isValidWebSocketParamType checks if a parameter type is valid for WebSocket functions
func (p *Parser) isValidWebSocketParamType(paramType string, validParamTypes map[string]bool) bool {
	// Check if it's in the explicit valid types
	if validParamTypes[paramType] {
		return true
	}

	// Allow custom pointer types (structs, interfaces)
	if strings.HasPrefix(paramType, "*") {
		return true
	}

	// Allow slice types []Type
	if strings.HasPrefix(paramType, "[]") {
		return true
	}

	// Allow map types map[K]V
	if strings.HasPrefix(paramType, "map[") {
		return true
	}

	// Allow chan types
	if strings.HasPrefix(paramType, "chan ") {
		return true
	}

	return false
}

// isValidWebSocketParamTypeWithDecorators checks if a parameter type is valid considering its decorators
func (p *Parser) isValidWebSocketParamTypeWithDecorators(param *core.ParameterNode, validParamTypes map[string]bool) bool {
	// Check for special decorator cases
	for _, decorator := range param.Decorators {
		switch decorator.Name {
		case "Exception":
			// @Exception() allows error type
			if param.Type == "error" {
				return true
			}
		case "EventName":
			// @EventName() allows string type
			if param.Type == "string" {
				return true
			}
		case "MessageBody":
			// @MessageBody() allows any type for message payload
			return true
		case "ConnectedSocket":
			// @ConnectedSocket() ONLY allows *WebSocketClient type for current connection
			if param.Type == "*WebSocketClient" {
				return true
			}
			// If it has @ConnectedSocket() decorator but wrong type, it's invalid
			return false
		case "MessageAck":
			// @MessageAck() ONLY allows *AckCallback type for acknowledgment callback
			if param.Type == "*AckCallback" {
				return true
			}
			// If it has @MessageAck() decorator but wrong type, it's invalid
			return false
		case "Rooms":
			// @Rooms() ONLY allows []string type for room collection
			if param.Type == "[]string" {
				return true
			}
			// If it has @Rooms() decorator but wrong type, it's invalid
			return false
		case "Namespace":
			// @Namespace() ONLY allows string type for namespace identifier
			if param.Type == "string" {
				return true
			}
			// If it has @Namespace() decorator but wrong type, it's invalid
			return false
		}
	}

	// Fall back to regular type validation
	return p.isValidWebSocketParamType(param.Type, validParamTypes)
}

// validateSubscribeMessageParameters validates parameters for @SubscribeMessage methods/functions
func (p *Parser) validateSubscribeMessageParameters(params []*core.ParameterNode, methodName string) {
	validParamTypes := map[string]bool{
		"*WebSocketClient":  true,
		"*AckCallback":      true,
		"string":            true,
		"interface{}":       true,
		"map[string]string": true,
		"*Session":          true,
		"*User":             true,
		"[]string":          true,
		"[]interface{}":     true,
		"int":               true,
		"bool":              true,
		"float64":           true,
	}

	for _, param := range params {
		if !p.isValidWebSocketParamType(param.Type, validParamTypes) {
			p.addError(fmt.Sprintf("invalid parameter type '%s' for @SubscribeMessage method '%s'. Expected: WebSocket types, slices, maps, or custom data types", param.Type, methodName))
		}
	}
}

// validateWebSocketGatewayField validates fields in WebSocket gateway
func (p *Parser) validateWebSocketGatewayField(field *core.FieldNode, gateway *core.WebSocketGatewayDeclaration) {
	if field == nil {
		return
	}

	// Check for dependency injection decorators
	hasInject := false
	for _, decorator := range field.Decorators {
		decoratorType := core.GetDecoratorType(decorator.Name)
		if decoratorType == core.InjectDecorator {
			hasInject = true
			break
		}
	}

	// Recommend @Inject for service dependencies
	if strings.Contains(field.Type, "Service") || strings.Contains(field.Type, "Repository") || strings.Contains(field.Type, "Logger") {
		if !hasInject {
			// This could be a warning rather than error
			p.addError(fmt.Sprintf("field '%s' in WebSocket gateway '%s' should use @Inject decorator for dependency injection", field.Name, gateway.Name))
		}
	}
}

// postProcessWebSocketValidation validates WebSocket declarations after decorators are attached
func (p *Parser) postProcessWebSocketValidation(decl core.GofaDeclaration) {
	switch d := decl.(type) {
	case *core.WebSocketGatewayDeclaration:
		p.validateWebSocketGateway(d)
	case *core.WebSocketFunctionDeclaration:
		p.validateWebSocketFunction(d)
	}
}

// skipComments skips comment tokens
func (p *Parser) skipComments() {
	for p.currToken.Type == COMMENT {
		p.nextToken()
	}
}

// attachDecoratorToDeclaration attaches a decorator to a declaration
func (p *Parser) attachDecoratorToDeclaration(decorator *core.DecoratorNode, decl core.GofaDeclaration) {
	if decl == nil || decorator == nil {
		return // Safety check to prevent nil pointer dereference
	}

	switch d := decl.(type) {
	case *core.ControllerDeclaration:
		if d != nil {
			d.Decorators = append(d.Decorators, decorator)
		}
	case *core.ServiceDeclaration:
		if d != nil {
			d.Decorators = append(d.Decorators, decorator)
		}
	case *core.ModuleDeclaration:
		if d != nil {
			d.Decorators = append(d.Decorators, decorator)
		}
	case *core.TestSuiteDeclaration:
		if d != nil {
			d.Decorators = append(d.Decorators, decorator)
		}
	case *core.WebSocketGatewayDeclaration:
		if d != nil {
			d.Decorators = append(d.Decorators, decorator)
		}
	case *core.WebSocketFunctionDeclaration:
		if d != nil {
			d.Decorators = append(d.Decorators, decorator)
		}
	}
}

// skipToNextDeclaration skips tokens until the next declaration
func (p *Parser) skipToNextDeclaration() {
	prevTokenType := p.currToken.Type
	p.nextToken()

	for p.currToken.Type != TYPE && p.currToken.Type != FUNC && p.currToken.Type != EOF && p.currToken.Type != DECORATOR {
		if p.currToken.Type == prevTokenType {
			p.nextToken()
			break
		}
		prevTokenType = p.currToken.Type
		p.nextToken()
	}
}

// isGoType checks if a token type is a Go built-in type
func isGoType(tokenType TokenType) bool {
	return tokenType >= GO_INT && tokenType <= GO_ERROR
}

// ParseGofaFile is the main entry point for parsing .gofa files
func ParseGofaFile(input string) (*core.GofaFile, error) {
	lexer := NewLexer(input)
	parser := NewParser(lexer)
	return parser.ParseFile()
}
