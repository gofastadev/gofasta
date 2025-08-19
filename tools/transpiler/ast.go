package transpiler

import (
	"go/ast"
	"go/token"
)

// GofaASTNode represents the base interface for all Gofasta AST nodes
type GofaASTNode interface {
	String() string
	Pos() token.Pos
}

// GofaFile represents a complete .gofa file
type GofaFile struct {
	Package      *ast.Ident          // package name
	Imports      []*ast.ImportSpec   // import declarations
	Decorators   []*DecoratorNode    // file-level decorators
	Declarations []GofaDeclaration   // type and function declarations
	Comments     []*ast.CommentGroup // comments
	Position     token.Pos
}

func (f *GofaFile) String() string {
	return "GofaFile"
}

func (f *GofaFile) Pos() token.Pos {
	return f.Position
}

// GofaDeclaration represents any top-level declaration in a .gofa file
type GofaDeclaration interface {
	GofaASTNode
	isDeclaration()
}

// DecoratorNode represents a decorator like @Controller("/api")
type DecoratorNode struct {
	Name     string          // decorator name (Controller, Get, etc.)
	Args     []DecoratorArg  // decorator arguments
	Target   GofaDeclaration // what this decorator applies to
	Position token.Pos
}

func (d *DecoratorNode) String() string {
	return "@" + d.Name
}

func (d *DecoratorNode) Pos() token.Pos {
	return d.Position
}

// DecoratorArg represents an argument to a decorator
type DecoratorArg struct {
	Key      string      // for named arguments like {path: "/api"}
	Value    interface{} // string, number, boolean, array, or object
	Position token.Pos
}

// ControllerDeclaration represents a controller class
type ControllerDeclaration struct {
	Name       string           // controller name
	Decorators []*DecoratorNode // @Controller, @UseGuards, etc.
	Fields     []*FieldNode     // injected dependencies
	Methods    []*MethodNode    // controller methods/routes
	Position   token.Pos
}

func (c *ControllerDeclaration) String() string {
	return "Controller: " + c.Name
}

func (c *ControllerDeclaration) Pos() token.Pos {
	return c.Position
}

func (c *ControllerDeclaration) isDeclaration() {}

// ServiceDeclaration represents a service/provider class
type ServiceDeclaration struct {
	Name       string           // service name
	Decorators []*DecoratorNode // @Injectable, etc.
	Fields     []*FieldNode     // injected dependencies
	Methods    []*MethodNode    // service methods
	Position   token.Pos
}

func (s *ServiceDeclaration) String() string {
	return "Service: " + s.Name
}

func (s *ServiceDeclaration) Pos() token.Pos {
	return s.Position
}

func (s *ServiceDeclaration) isDeclaration() {}

// ModuleDeclaration represents a module
type ModuleDeclaration struct {
	Name       string           // module name
	Decorators []*DecoratorNode // @Module decorator
	Position   token.Pos
}

func (m *ModuleDeclaration) String() string {
	return "Module: " + m.Name
}

func (m *ModuleDeclaration) Pos() token.Pos {
	return m.Position
}

func (m *ModuleDeclaration) isDeclaration() {}

// FieldNode represents a field with dependency injection
type FieldNode struct {
	Name       string           // field name
	Type       string           // field type
	Tag        string           // struct tag for injection
	Decorators []*DecoratorNode // field decorators if any
	Position   token.Pos
}

func (f *FieldNode) String() string {
	return f.Name + " " + f.Type
}

func (f *FieldNode) Pos() token.Pos {
	return f.Position
}

// MethodNode represents a method with possible route decorators
type MethodNode struct {
	Name       string           // method name
	Params     []*ParameterNode // method parameters
	ReturnType string           // return type
	Body       []ast.Stmt       // method body statements
	Decorators []*DecoratorNode // @Get, @Post, etc.
	Position   token.Pos
}

func (m *MethodNode) String() string {
	return "Method: " + m.Name
}

func (m *MethodNode) Pos() token.Pos {
	return m.Position
}

// ParameterNode represents a method parameter with possible decorators
type ParameterNode struct {
	Name       string           // parameter name
	Type       string           // parameter type
	Decorators []*DecoratorNode // @Body, @Param, @Query, etc.
	Position   token.Pos
}

func (p *ParameterNode) String() string {
	return p.Name + " " + p.Type
}

func (p *ParameterNode) Pos() token.Pos {
	return p.Position
}

// DecoratorType enumeration for different decorator types
type DecoratorType int

const (
	// Class-level decorators
	ControllerDecorator DecoratorType = iota
	ServiceDecorator
	ModuleDecorator
	InjectableDecorator

	// Method-level decorators
	GetDecorator
	PostDecorator
	PutDecorator
	DeleteDecorator
	PatchDecorator
	OptionsDecorator
	HeadDecorator

	// Cross-cutting decorators
	UseGuardsDecorator
	UseMiddlewareDecorator
	UsePipesDecorator
	UseFiltersDecorator
	UseInterceptorsDecorator

	// Parameter decorators
	BodyDecorator
	ParamDecorator
	QueryDecorator
	HeadersDecorator
	RequestDecorator
	ResponseDecorator
	SessionDecorator
	IpDecorator
	HostParamDecorator

	// Other decorators
	HttpCodeDecorator
	VersionDecorator
	RolesDecorator
	CacheDecorator
	ThrottleDecorator

	// Custom decorators
	CustomDecorator
)

// DecoratorTypeMap maps decorator names to their types
var DecoratorTypeMap = map[string]DecoratorType{
	"Controller":      ControllerDecorator,
	"Injectable":      InjectableDecorator,
	"Module":          ModuleDecorator,
	"Get":             GetDecorator,
	"Post":            PostDecorator,
	"Put":             PutDecorator,
	"Delete":          DeleteDecorator,
	"Patch":           PatchDecorator,
	"Options":         OptionsDecorator,
	"Head":            HeadDecorator,
	"UseGuards":       UseGuardsDecorator,
	"UseMiddleware":   UseMiddlewareDecorator,
	"UsePipes":        UsePipesDecorator,
	"UseFilters":      UseFiltersDecorator,
	"UseInterceptors": UseInterceptorsDecorator,
	"Body":            BodyDecorator,
	"Param":           ParamDecorator,
	"Query":           QueryDecorator,
	"Headers":         HeadersDecorator,
	"Req":             RequestDecorator,
	"Res":             ResponseDecorator,
	"Session":         SessionDecorator,
	"Ip":              IpDecorator,
	"HostParam":       HostParamDecorator,
	"HttpCode":        HttpCodeDecorator,
	"Version":         VersionDecorator,
	"Roles":           RolesDecorator,
	"Cache":           CacheDecorator,
	"Throttle":        ThrottleDecorator,
}

// GetDecoratorType returns the decorator type for a given name
func GetDecoratorType(name string) DecoratorType {
	if decoratorType, exists := DecoratorTypeMap[name]; exists {
		return decoratorType
	}
	return CustomDecorator
}

// IsRouteDecorator checks if a decorator type is a route decorator
func IsRouteDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= GetDecorator && decoratorType <= HeadDecorator
}

// IsParameterDecorator checks if a decorator type is a parameter decorator
func IsParameterDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= BodyDecorator && decoratorType <= HostParamDecorator
}

// IsClassDecorator checks if a decorator type is a class-level decorator
func IsClassDecorator(decoratorType DecoratorType) bool {
	return decoratorType <= InjectableDecorator
}

// IsCrossCuttingDecorator checks if a decorator type is a cross-cutting concern decorator
func IsCrossCuttingDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= UseGuardsDecorator && decoratorType <= UseInterceptorsDecorator
}

// Visitor interface for traversing the AST
type Visitor interface {
	Visit(node GofaASTNode) Visitor
}

// Walk traverses an AST in depth-first order
func Walk(v Visitor, node GofaASTNode) {
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case *GofaFile:
		for _, decl := range n.Declarations {
			Walk(v, decl)
		}
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}

	case *ControllerDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, field := range n.Fields {
			Walk(v, field)
		}
		for _, method := range n.Methods {
			Walk(v, method)
		}

	case *ServiceDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, field := range n.Fields {
			Walk(v, field)
		}
		for _, method := range n.Methods {
			Walk(v, method)
		}

	case *ModuleDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}

	case *MethodNode:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, param := range n.Params {
			Walk(v, param)
		}

	case *ParameterNode:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}

	case *FieldNode:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
	}
}
