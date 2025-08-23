package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// Parser represents the parser state
type Parser struct {
	lexer      *Lexer
	currToken  Token
	peekToken  Token
	errors     []string
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
func (p *Parser) ParseFile() (*GofaFile, error) {
	file := &GofaFile{
		Position:     p.currToken.Position,
		Declarations: []GofaDeclaration{},
		Decorators:   []*DecoratorNode{},
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
			var decorators []*DecoratorNode
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
						if controller, ok := file.Declarations[len(file.Declarations)-1].(*ControllerDeclaration); ok {
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
						} else if testSuite, ok := file.Declarations[len(file.Declarations)-1].(*TestSuiteDeclaration); ok {
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
						}
					}
				}
				
				// Parse the declaration and attach all decorators
				decl := p.parseDeclarationWithDecorators(decorators)
				if decl != nil {
					for _, decorator := range decorators {
						p.attachDecoratorToDeclaration(decorator, decl)
					}
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
			// Note: parseDeclaration() handles its own token consumption, including for functions
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
func (p *Parser) parseDeclaration() GofaDeclaration {
	return p.parseDeclarationWithDecorators(nil)
}

// parseDeclarationWithDecorators parses a declaration with access to decorators
func (p *Parser) parseDeclarationWithDecorators(decorators []*DecoratorNode) GofaDeclaration {
	switch p.currToken.Type {
	case TYPE:
		return p.parseTypeDeclarationWithDecorators(decorators)
	case FUNC:
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
func (p *Parser) parseTypeDeclarationWithDecorators(decorators []*DecoratorNode) GofaDeclaration {
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
	
	// Check for explicit @TestSuite() decorator first
	if p.hasTestSuiteDecorator(decorators) {
		return p.parseTestSuiteDeclaration(typeName)
	}
	
	// Check for explicit @Factory() decorator
	if p.hasFactoryDecorator(decorators) {
		return p.parseFactoryDeclaration(typeName, decorators)
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

// hasTestSuiteDecorator checks if decorators contain @TestSuite() decorator
func (p *Parser) hasTestSuiteDecorator(decorators []*DecoratorNode) bool {
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

// hasFactoryDecorator checks if decorators contain @Factory() decorator
func (p *Parser) hasFactoryDecorator(decorators []*DecoratorNode) bool {
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

// parseControllerDeclaration parses a controller declaration
func (p *Parser) parseControllerDeclaration(name string) *ControllerDeclaration {
	controller := &ControllerDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*DecoratorNode{},
		Fields:     []*FieldNode{},
		Methods:    []*MethodNode{},
	}
	
	// Parse fields (including field decorators)
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			// Parse field decorators
			var decorators []*DecoratorNode
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
					// Attach decorators to field
					field.Decorators = decorators
					controller.Fields = append(controller.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				// Invalid field after decorators
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				controller.Fields = append(controller.Fields, field)
			} else {
				// If field parsing failed, advance token to avoid infinite loop
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}
	
	if !p.expectToken(RBRACE) {
		return nil
	}
	
	// Parse standalone functions immediately following the struct as methods
	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			controller.Methods = append(controller.Methods, method)
		} else {
			// If method parsing failed, break to avoid infinite loop
			break
		}
	}
	
	return controller
}

// parseServiceDeclaration parses a service declaration
func (p *Parser) parseServiceDeclaration(name string) *ServiceDeclaration {
	service := &ServiceDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*DecoratorNode{},
		Fields:     []*FieldNode{},
		Methods:    []*MethodNode{},
	}
	
	// Parse fields (including field decorators)
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			// Parse field decorators
			var decorators []*DecoratorNode
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
					// Attach decorators to field
					field.Decorators = decorators
					service.Fields = append(service.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				// Invalid field after decorators
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				service.Fields = append(service.Fields, field)
			} else {
				// If field parsing failed, advance token to avoid infinite loop
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}
	
	if !p.expectToken(RBRACE) {
		return nil
	}
	
	// Parse standalone functions immediately following the struct as methods
	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			service.Methods = append(service.Methods, method)
		} else {
			// If method parsing failed, break to avoid infinite loop
			break
		}
	}
	
	return service
}

// parseModuleDeclaration parses a module declaration
func (p *Parser) parseModuleDeclaration(name string) *ModuleDeclaration {
	module := &ModuleDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*DecoratorNode{},
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
func (p *Parser) parseTestSuiteDeclaration(name string) *TestSuiteDeclaration {
	testSuite := &TestSuiteDeclaration{
		Name:       name,
		Position:   p.currToken.Position,
		Decorators: []*DecoratorNode{},
		Fields:     []*FieldNode{},
		Methods:    []*MethodNode{},
	}
	
	// Parse fields (including field decorators for mocks and dependencies)
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			// Parse field decorators
			var decorators []*DecoratorNode
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
					// Attach decorators to field
					field.Decorators = decorators
					testSuite.Fields = append(testSuite.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				// Invalid field after decorators
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				testSuite.Fields = append(testSuite.Fields, field)
			} else {
				// If field parsing failed, advance token to avoid infinite loop
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}
	
	if !p.expectToken(RBRACE) {
		return nil
	}
	
	// Parse standalone functions immediately following the struct as methods
	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			testSuite.Methods = append(testSuite.Methods, method)
		} else {
			// If method parsing failed, break to avoid infinite loop
			break
		}
	}
	
	return testSuite
}

// parseFactoryDeclaration parses a factory declaration
func (p *Parser) parseFactoryDeclaration(name string, decorators []*DecoratorNode) *FactoryDeclaration {
	// Extract target type from factory name (e.g., "UserFactory" -> "User")
	targetType := name
	if strings.HasSuffix(name, "Factory") {
		targetType = strings.TrimSuffix(name, "Factory")
	}

	factory := &FactoryDeclaration{
		Name:       name,
		TargetType: targetType,
		Position:   p.currToken.Position,
		Decorators: decorators,
		Fields:     []*FieldNode{},
		Methods:    []*MethodNode{},
	}
	
	// Parse fields (factory configuration and dependencies)
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == DECORATOR {
			// Parse field decorators
			var fieldDecorators []*DecoratorNode
			for p.currToken.Type == DECORATOR {
				decorator := p.parseDecorator()
				if decorator != nil {
					fieldDecorators = append(fieldDecorators, decorator)
				} else {
					p.nextToken()
					break
				}
			}
			
			// Parse the field after decorators
			if p.currToken.Type == IDENT {
				field := p.parseField()
				if field != nil {
					// Attach decorators to field
					field.Decorators = fieldDecorators
					factory.Fields = append(factory.Fields, field)
				} else {
					p.nextToken()
				}
			} else {
				// Invalid field after decorators
				p.addError("expected field after decorator")
				p.nextToken()
			}
		} else if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				factory.Fields = append(factory.Fields, field)
			} else {
				// If field parsing failed, advance token to avoid infinite loop
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}
	
	if !p.expectToken(RBRACE) {
		return nil
	}
	
	// Parse standalone functions immediately following the struct as methods
	for p.currToken.Type == FUNC {
		method := p.parseMethod()
		if method != nil {
			factory.Methods = append(factory.Methods, method)
		}
	}
	
	return factory
}

// parseField parses a struct field with possible injection tags
func (p *Parser) parseField() *FieldNode {
	if p.currToken.Type != IDENT {
		return nil
	}
	
	field := &FieldNode{
		Name:       p.currToken.Literal,
		Position:   p.currToken.Position,
		Decorators: []*DecoratorNode{},
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

// parseMethod parses a method declaration (standalone function)
func (p *Parser) parseMethod() *MethodNode {
	if !p.expectToken(FUNC) {
		return nil
	}
	
	// Check if this has a receiver (method) or is a standalone function
	if p.currToken.Type == LPAREN {
		// Might be a receiver, check if it's followed by identifier and type
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
	
	method := &MethodNode{
		Name:       p.currToken.Literal,
		Position:   p.currToken.Position,
		Params:     []*ParameterNode{},
		Decorators: []*DecoratorNode{},
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
			
			// Handle comma separation properly
			if p.currToken.Type == COMMA {
				p.nextToken()
			} else if p.currToken.Type != RPAREN {
				// If not a comma or closing paren, we might be stuck
				p.addError("expected ',' or ')' in parameter list")
				break
			}
		}
		if !p.expectToken(RPAREN) {
			return nil
		}
	}
	
	// Parse return type if present (can be complex: (result []string, err error))
	if p.currToken.Type != LBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == LPAREN {
			// Complex return type with named returns: (result []string, err error)
			p.nextToken() // consume (
			var returnTypes []string
			for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
				if p.currToken.Type == IDENT {
					// Skip parameter name
					p.nextToken()
				}
				// Parse type
				if p.currToken.Type != COMMA && p.currToken.Type != RPAREN {
					returnTypes = append(returnTypes, p.parseType())
				}
				// Skip comma
				if p.currToken.Type == COMMA {
					p.nextToken()
				}
			}
			if p.currToken.Type == RPAREN {
				p.nextToken() // consume )
			}
			if len(returnTypes) > 0 {
				method.ReturnType = returnTypes[0] // Use first return type for simplicity
			}
		} else {
			// Simple return type
			method.ReturnType = p.parseType()
		}
	}
	
	// Parse method body
	if p.currToken.Type == LBRACE {
		p.parseBlockStatement() // Just skip the body for now
	}
	
	return method
}

// parseParameter parses a method parameter with possible decorators
func (p *Parser) parseParameter() *ParameterNode {
	param := &ParameterNode{
		Position:   p.currToken.Position,
		Decorators: []*DecoratorNode{},
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
	
	// Handle multiple pointer types (**Type)
	for p.currToken.Type == MULTIPLY {
		typeStr.WriteString("*")
		p.nextToken()
	}
	
	// Handle slice types including nested slices ([][]Type)
	for p.currToken.Type == LBRACKET {
		typeStr.WriteString("[")
		p.nextToken()
		if p.currToken.Type == RBRACKET {
			typeStr.WriteString("]")
			p.nextToken()
		}
	}
	
	// Handle channel types (chan Type)
	if p.currToken.Type == GO_CHAN {
		typeStr.WriteString("chan")
		p.nextToken()
		// Handle directional channels - simplified for now
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
	
	// Handle map types (map[KeyType]ValueType)
	if p.currToken.Type == GO_MAP {
		typeStr.WriteString("map[")
		p.nextToken()
		// Should be at the LBRACKET after "map"
		if p.currToken.Type == LBRACKET {
			p.nextToken() // consume the [
		}
		// Parse key type
		keyType := p.parseType()
		typeStr.WriteString(keyType)
		// Should be at RBRACKET
		if p.currToken.Type == RBRACKET {
			typeStr.WriteString("]")
			p.nextToken() // consume the ]
		}
		// Parse value type
		valueType := p.parseType()
		typeStr.WriteString(valueType)
		return typeStr.String()
	}
	
	// Handle function types (func(Type) ReturnType)
	if p.currToken.Type == FUNC {
		typeStr.WriteString("func")
		p.nextToken()
		
		// Parse parameters
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
		
		// Parse return type if present
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
		// Skip the {} part
		if p.currToken.Type == LBRACE {
			p.nextToken()
			if p.currToken.Type == RBRACE {
				p.nextToken()
			}
		}
		return typeStr.String()
	}
	
	// Handle anonymous struct types (struct { ... })
	if p.currToken.Type == STRUCT {
		typeStr.WriteString("struct")
		p.nextToken()
		
		if p.currToken.Type == LBRACE {
			typeStr.WriteString(" {")
			p.nextToken()
			
			// Parse struct fields recursively
			for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
				// Skip whitespace and newlines by advancing
				if p.currToken.Type == IDENT {
					// Field name
					typeStr.WriteString("\n\t\t")
					typeStr.WriteString(p.currToken.Literal)
					p.nextToken()
					
					// Field type
					typeStr.WriteString(" ")
					fieldType := p.parseType()
					typeStr.WriteString(fieldType)
					
					// Optional struct tag
					if p.currToken.Type == STRING {
						typeStr.WriteString(" ")
						typeStr.WriteString(p.currToken.Literal)
						p.nextToken()
					}
				} else {
					// Skip unexpected tokens to avoid infinite loop
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
func (p *Parser) parseDecorator() *DecoratorNode {
	if !p.expectToken(DECORATOR) {
		return nil
	}
	
	if p.currToken.Type != IDENT {
		p.addError("expected decorator name")
		return nil
	}
	
	decorator := &DecoratorNode{
		Name:     p.currToken.Literal,
		Position: p.currToken.Position,
		Args:     []DecoratorArg{},
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
				// If arg parsing failed, advance token to avoid infinite loop
				p.nextToken()
			}
			
			if p.currToken.Type == COMMA {
				p.nextToken()
			} else if p.currToken.Type != RPAREN {
				// If not comma or rparen, we might be stuck
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
func (p *Parser) parseDecoratorArg() *DecoratorArg {
	arg := &DecoratorArg{
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
		// Could be a named argument or reference
		arg.Key = p.currToken.Literal
		arg.Value = p.currToken.Literal // Default to the identifier value
		p.nextToken()
		if p.currToken.Type == COLON {
			p.nextToken()
			// Parse simple values only (avoid recursion)
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
				// Parse array values like ["item1", "item2"]
				arrayValues := []interface{}{}
				p.nextToken() // consume [
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
						p.nextToken() // skip unknown tokens
					}
					if p.currToken.Type == COMMA {
						p.nextToken()
					}
				}
				if p.currToken.Type == RBRACKET {
					p.nextToken() // consume ]
				}
				arg.Value = arrayValues
			default:
				arg.Value = p.currToken.Literal
				p.nextToken()
			}
		}
	case LBRACE:
		// Parse object literal like { key: value, key2: [array] }
		objectValue := make(map[string]interface{})
		p.nextToken() // consume {
		
		for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
			// Parse key
			if p.currToken.Type != IDENT {
				p.nextToken() // skip non-ident
				continue
			}
			key := p.currToken.Literal
			p.nextToken()
			
			// Expect colon
			if p.currToken.Type != COLON {
				p.nextToken() // skip if no colon
				continue
			}
			p.nextToken() // consume :
			
			// Parse value
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
				// Parse array value
				arrayValues := []interface{}{}
				p.nextToken() // consume [
				for p.currToken.Type != RBRACKET && p.currToken.Type != EOF {
					switch p.currToken.Type {
					case STRING:
						arrayValues = append(arrayValues, p.currToken.Literal)
						p.nextToken()
					case IDENT:
						arrayValues = append(arrayValues, p.currToken.Literal)
						p.nextToken()
					default:
						p.nextToken() // skip
					}
					if p.currToken.Type == COMMA {
						p.nextToken()
					}
				}
				if p.currToken.Type == RBRACKET {
					p.nextToken() // consume ]
				}
				objectValue[key] = arrayValues
			default:
				objectValue[key] = p.currToken.Literal
				p.nextToken()
			}
			
			// Handle comma
			if p.currToken.Type == COMMA {
				p.nextToken()
			}
		}
		
		if p.currToken.Type == RBRACE {
			p.nextToken() // consume }
		}
		arg.Value = objectValue
	default:
		p.addError(fmt.Sprintf("unexpected decorator argument type %d at line %d", 
			int(p.currToken.Type), p.currToken.Line))
		p.nextToken() // Advance to avoid infinite loop
		return nil
	}
	
	return arg
}

// parseFunctionDeclaration parses a standalone function (not method)
func (p *Parser) parseFunctionDeclaration() GofaDeclaration {
	// Skip the func keyword
	if !p.expectToken(FUNC) {
		return nil
	}
	
	// Skip function name
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
	
	// Skip return type if present
	for p.currToken.Type != LBRACE && p.currToken.Type != EOF {
		p.nextToken()
	}
	
	// Skip function body if present
	if p.currToken.Type == LBRACE {
		p.parseBlockStatement()
	}
	
	// For now, we don't create a declaration for standalone functions
	return nil
}

// parseBlockStatement parses a block statement (method body)
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

// Helper functions

// skipComments skips comment tokens
func (p *Parser) skipComments() {
	for p.currToken.Type == COMMENT {
		p.nextToken()
	}
}


// attachDecoratorToDeclaration attaches a decorator to a declaration
func (p *Parser) attachDecoratorToDeclaration(decorator *DecoratorNode, decl GofaDeclaration) {
	switch d := decl.(type) {
	case *ControllerDeclaration:
		d.Decorators = append(d.Decorators, decorator)
	case *ServiceDeclaration:
		d.Decorators = append(d.Decorators, decorator)
	case *ModuleDeclaration:
		d.Decorators = append(d.Decorators, decorator)
	case *TestSuiteDeclaration:
		d.Decorators = append(d.Decorators, decorator)
	}
}

// skipToNextDeclaration skips tokens until the next declaration
func (p *Parser) skipToNextDeclaration() {
	// Skip current token
	prevTokenType := p.currToken.Type
	p.nextToken()
	
	// Continue skipping until we find a declaration token
	for p.currToken.Type != TYPE && p.currToken.Type != FUNC && p.currToken.Type != EOF && p.currToken.Type != DECORATOR {
		// Prevent infinite loop by ensuring token actually advances
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
func ParseGofaFile(input string) (*GofaFile, error) {
	lexer := NewLexer(input)
	parser := NewParser(lexer)
	return parser.ParseFile()
}