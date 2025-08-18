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
			decorator := p.parseDecorator()
			if decorator != nil {
				// Check if this decorator applies to the next declaration
				if p.isNextTokenDeclaration() {
					// Parse the declaration and attach the decorator
					decl := p.parseDeclaration()
					if decl != nil {
						p.attachDecoratorToDeclaration(decorator, decl)
						file.Declarations = append(file.Declarations, decl)
					}
				} else {
					// File-level decorator
					file.Decorators = append(file.Decorators, decorator)
				}
			}
		} else {
			decl := p.parseDeclaration()
			if decl != nil {
				file.Declarations = append(file.Declarations, decl)
			}
		}
	}
	
	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parsing errors: %s", strings.Join(p.errors, "; "))
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
	switch p.currToken.Type {
	case TYPE:
		return p.parseTypeDeclaration()
	case FUNC:
		return p.parseFunctionDeclaration()
	default:
		p.addError(fmt.Sprintf("unexpected token %s", tokenTypeNames[p.currToken.Type]))
		p.nextToken()
		return nil
	}
}

// parseTypeDeclaration parses a type declaration (controller, service, module)
func (p *Parser) parseTypeDeclaration() GofaDeclaration {
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
	
	// Determine declaration type based on naming convention or decorators
	if strings.HasSuffix(typeName, "Controller") {
		return p.parseControllerDeclaration(typeName)
	} else if strings.HasSuffix(typeName, "Service") {
		return p.parseServiceDeclaration(typeName)
	} else if strings.HasSuffix(typeName, "Module") {
		return p.parseModuleDeclaration(typeName)
	}
	
	// Default to service declaration
	return p.parseServiceDeclaration(typeName)
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
	
	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				controller.Fields = append(controller.Fields, field)
			}
		} else {
			p.nextToken()
		}
	}
	
	if !p.expectToken(RBRACE) {
		return nil
	}
	
	// Parse methods
	for p.currToken.Type == FUNC && p.peekToken.Type == LPAREN {
		method := p.parseMethod()
		if method != nil {
			controller.Methods = append(controller.Methods, method)
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
	
	// Parse fields
	for p.currToken.Type != RBRACE && p.currToken.Type != EOF {
		if p.currToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				service.Fields = append(service.Fields, field)
			}
		} else {
			p.nextToken()
		}
	}
	
	if !p.expectToken(RBRACE) {
		return nil
	}
	
	// Parse methods
	for p.currToken.Type == FUNC && p.peekToken.Type == LPAREN {
		method := p.parseMethod()
		if method != nil {
			service.Methods = append(service.Methods, method)
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

// parseMethod parses a method declaration
func (p *Parser) parseMethod() *MethodNode {
	if !p.expectToken(FUNC) {
		return nil
	}
	
	// Parse receiver
	if !p.expectToken(LPAREN) {
		return nil
	}
	
	// Skip receiver type
	for p.currToken.Type != RPAREN && p.currToken.Type != EOF {
		p.nextToken()
	}
	
	if !p.expectToken(RPAREN) {
		return nil
	}
	
	// Parse method name
	if p.currToken.Type != IDENT {
		p.addError("expected method name")
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
			
			if p.currToken.Type == COMMA {
				p.nextToken()
			}
		}
		if !p.expectToken(RPAREN) {
			return nil
		}
	}
	
	// Parse return type
	if p.currToken.Type != LBRACE {
		method.ReturnType = p.parseType()
	}
	
	// Parse method body
	if p.currToken.Type == LBRACE {
		p.parseBlockStatement() // Just skip the body for now
	}
	
	return method
}

// parseParameter parses a method parameter with possible decorators
func (p *Parser) parseParameter() *ParameterNode {
	if p.currToken.Type != IDENT {
		return nil
	}
	
	param := &ParameterNode{
		Name:       p.currToken.Literal,
		Position:   p.currToken.Position,
		Decorators: []*DecoratorNode{},
	}
	p.nextToken()
	
	param.Type = p.parseType()
	
	return param
}

// parseType parses a type expression
func (p *Parser) parseType() string {
	var typeStr strings.Builder
	
	// Handle pointer types
	if p.currToken.Type == MULTIPLY {
		typeStr.WriteString("*")
		p.nextToken()
	}
	
	// Handle slice types
	if p.currToken.Type == LBRACKET {
		typeStr.WriteString("[")
		p.nextToken()
		if p.currToken.Type == RBRACKET {
			typeStr.WriteString("]")
			p.nextToken()
		}
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
			}
			
			if p.currToken.Type == COMMA {
				p.nextToken()
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
		p.nextToken()
		if p.currToken.Type == COLON {
			p.nextToken()
			// Parse the value
			subArg := p.parseDecoratorArg()
			if subArg != nil {
				arg.Value = subArg.Value
			}
		}
	default:
		p.addError(fmt.Sprintf("unexpected decorator argument type %s", 
			tokenTypeNames[p.currToken.Type]))
		return nil
	}
	
	return arg
}

// parseFunctionDeclaration parses a standalone function (not method)
func (p *Parser) parseFunctionDeclaration() GofaDeclaration {
	// For now, just skip standalone functions
	p.skipToNextDeclaration()
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

// isNextTokenDeclaration checks if the next token starts a declaration
func (p *Parser) isNextTokenDeclaration() bool {
	return p.peekToken.Type == TYPE || p.peekToken.Type == FUNC
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
	}
}

// skipToNextDeclaration skips tokens until the next declaration
func (p *Parser) skipToNextDeclaration() {
	for p.currToken.Type != TYPE && p.currToken.Type != FUNC && p.currToken.Type != EOF {
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