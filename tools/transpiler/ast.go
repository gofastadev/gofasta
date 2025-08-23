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

// TestSuiteDeclaration represents a test suite class
type TestSuiteDeclaration struct {
	Name       string           // test suite name
	Decorators []*DecoratorNode // @TestSuite, @HTTPTest, @DatabaseTest, etc.
	Fields     []*FieldNode     // injected dependencies and mocks
	Methods    []*MethodNode    // test methods with @Test, @BeforeEach, etc.
	Position   token.Pos
}

func (ts *TestSuiteDeclaration) String() string {
	return "TestSuite: " + ts.Name
}

func (ts *TestSuiteDeclaration) Pos() token.Pos {
	return ts.Position
}

func (ts *TestSuiteDeclaration) isDeclaration() {}

// FactoryDeclaration represents a test data factory class
type FactoryDeclaration struct {
	Name       string           // factory name (e.g., "UserFactory")
	TargetType string           // target struct type (e.g., "User")  
	Decorators []*DecoratorNode // @Factory decorator
	Fields     []*FieldNode     // factory configuration fields
	Methods    []*MethodNode    // Build method and traits
	Position   token.Pos
}

func (f *FactoryDeclaration) String() string {
	return "Factory: " + f.Name
}

func (f *FactoryDeclaration) Pos() token.Pos {
	return f.Position
}

func (f *FactoryDeclaration) isDeclaration() {}

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

	// Error handling decorators
	CatchDecorator

	// Dependency injection decorators
	InjectDecorator
	ProviderDecorator
	ScopeDecorator

	// Other decorators
	HttpCodeDecorator
	RedirectDecorator
	HeaderDecorator
	VersionDecorator
	RolesDecorator
	CacheDecorator
	ThrottleDecorator

	// Validation decorators - Type validation
	IsStringDecorator
	IsNumberDecorator
	IsIntDecorator
	IsFloatDecorator
	IsBooleanDecorator
	IsArrayDecorator
	IsObjectDecorator
	IsDateDecorator
	IsUUIDDecorator

	// Validation decorators - Format validation
	IsEmailDecorator
	IsURLDecorator
	IsIPDecorator
	IsJSONDecorator
	IsAlphaDecorator
	IsAlphanumericDecorator
	IsNumericDecorator
	IsHexColorDecorator
	IsPhoneNumberDecorator
	IsCreditCardDecorator
	IsISBNDecorator
	IsBase64Decorator

	// Validation decorators - Range and length
	MinDecorator
	MaxDecorator
	LengthDecorator
	MinLengthDecorator
	MaxLengthDecorator
	ArrayMinSizeDecorator
	ArrayMaxSizeDecorator
	ArrayNotEmptyDecorator

	// Validation decorators - Content validation
	IsNotEmptyDecorator
	IsEmptyDecorator
	IsOptionalDecorator
	IsDefinedDecorator
	NotEqualsDecorator
	EqualsDecorator
	ContainsDecorator
	NotContainsDecorator
	IsInDecorator
	IsNotInDecorator

	// Validation decorators - Pattern and custom
	MatchesDecorator
	IsLowercaseDecorator
	IsUppercaseDecorator
	ValidateNestedDecorator
	ValidateIfDecorator
	CustomValidatorDecorator

	// Validation decorators - Business logic
	IsPositiveDecorator
	IsNegativeDecorator
	IsPastDateDecorator
	IsFutureDateDecorator

	// Testing decorators
	TestSuiteDecorator
	TestDecorator
	BeforeEachDecorator
	AfterEachDecorator
	BeforeAllDecorator
	AfterAllDecorator
	MockDecorator
	FactoryDecorator
	TraitDecorator
	TestModuleDecorator
	HTTPTestDecorator
	DatabaseTestDecorator

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
	"Catch":           CatchDecorator,
	"Inject":          InjectDecorator,
	"Provider":        ProviderDecorator,
	"Scope":           ScopeDecorator,
	"HttpCode":        HttpCodeDecorator,
	"Redirect":        RedirectDecorator,
	"Header":          HeaderDecorator,
	"Version":         VersionDecorator,
	"Roles":           RolesDecorator,
	"Cache":           CacheDecorator,
	"Throttle":        ThrottleDecorator,
	
	// Type validation decorators
	"IsString":        IsStringDecorator,
	"IsNumber":        IsNumberDecorator,
	"IsInt":           IsIntDecorator,
	"IsFloat":         IsFloatDecorator,
	"IsBoolean":       IsBooleanDecorator,
	"IsArray":         IsArrayDecorator,
	"IsObject":        IsObjectDecorator,
	"IsDate":          IsDateDecorator,
	"IsUUID":          IsUUIDDecorator,
	
	// Format validation decorators
	"IsEmail":         IsEmailDecorator,
	"IsURL":           IsURLDecorator,
	"IsIP":            IsIPDecorator,
	"IsJSON":          IsJSONDecorator,
	"IsAlpha":         IsAlphaDecorator,
	"IsAlphanumeric":  IsAlphanumericDecorator,
	"IsNumeric":       IsNumericDecorator,
	"IsHexColor":      IsHexColorDecorator,
	"IsPhoneNumber":   IsPhoneNumberDecorator,
	"IsCreditCard":    IsCreditCardDecorator,
	"IsISBN":          IsISBNDecorator,
	"IsBase64":        IsBase64Decorator,
	
	// Range and length validation decorators
	"Min":             MinDecorator,
	"Max":             MaxDecorator,
	"Length":          LengthDecorator,
	"MinLength":       MinLengthDecorator,
	"MaxLength":       MaxLengthDecorator,
	"ArrayMinSize":    ArrayMinSizeDecorator,
	"ArrayMaxSize":    ArrayMaxSizeDecorator,
	"ArrayNotEmpty":   ArrayNotEmptyDecorator,
	
	// Content validation decorators
	"IsNotEmpty":      IsNotEmptyDecorator,
	"IsEmpty":         IsEmptyDecorator,
	"IsOptional":      IsOptionalDecorator,
	"IsDefined":       IsDefinedDecorator,
	"NotEquals":       NotEqualsDecorator,
	"Equals":          EqualsDecorator,
	"Contains":        ContainsDecorator,
	"NotContains":     NotContainsDecorator,
	"IsIn":            IsInDecorator,
	"IsNotIn":         IsNotInDecorator,
	
	// Pattern and custom validation decorators
	"Matches":         MatchesDecorator,
	"IsLowercase":     IsLowercaseDecorator,
	"IsUppercase":     IsUppercaseDecorator,
	"ValidateNested":  ValidateNestedDecorator,
	"ValidateIf":      ValidateIfDecorator,
	"Custom":          CustomValidatorDecorator,
	
	// Business logic validation decorators
	"IsPositive":      IsPositiveDecorator,
	"IsNegative":      IsNegativeDecorator,
	"IsPastDate":      IsPastDateDecorator,
	"IsFutureDate":    IsFutureDateDecorator,
	
	// Testing decorators
	"TestSuite":       TestSuiteDecorator,
	"Test":            TestDecorator,
	"BeforeEach":      BeforeEachDecorator,
	"AfterEach":       AfterEachDecorator,
	"BeforeAll":       BeforeAllDecorator,
	"AfterAll":        AfterAllDecorator,
	"Mock":            MockDecorator,
	"Factory":         FactoryDecorator,
	"Trait":           TraitDecorator,
	"TestModule":      TestModuleDecorator,
	"HTTPTest":        HTTPTestDecorator,
	"DatabaseTest":    DatabaseTestDecorator,
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

// IsErrorHandlingDecorator checks if a decorator type is an error handling decorator
func IsErrorHandlingDecorator(decoratorType DecoratorType) bool {
	return decoratorType == CatchDecorator
}

// IsDependencyInjectionDecorator checks if a decorator type is a dependency injection decorator
func IsDependencyInjectionDecorator(decoratorType DecoratorType) bool {
	return decoratorType == InjectDecorator || decoratorType == ProviderDecorator || decoratorType == ScopeDecorator
}

// IsValidationDecorator checks if a decorator type is a validation decorator
func IsValidationDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= IsStringDecorator && decoratorType <= IsFutureDateDecorator
}

// IsTypeValidationDecorator checks if a decorator type is a type validation decorator
func IsTypeValidationDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= IsStringDecorator && decoratorType <= IsUUIDDecorator
}

// IsFormatValidationDecorator checks if a decorator type is a format validation decorator
func IsFormatValidationDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= IsEmailDecorator && decoratorType <= IsBase64Decorator
}

// IsRangeValidationDecorator checks if a decorator type is a range/length validation decorator
func IsRangeValidationDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= MinDecorator && decoratorType <= ArrayNotEmptyDecorator
}

// IsContentValidationDecorator checks if a decorator type is a content validation decorator
func IsContentValidationDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= IsNotEmptyDecorator && decoratorType <= IsNotInDecorator
}

// IsPatternValidationDecorator checks if a decorator type is a pattern validation decorator
func IsPatternValidationDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= MatchesDecorator && decoratorType <= CustomValidatorDecorator
}

// IsBusinessLogicValidationDecorator checks if a decorator type is a business logic validation decorator
func IsBusinessLogicValidationDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= IsPositiveDecorator && decoratorType <= IsFutureDateDecorator
}

// IsTestingDecorator checks if a decorator type is a testing decorator
func IsTestingDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= TestSuiteDecorator && decoratorType <= DatabaseTestDecorator
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

	case *TestSuiteDeclaration:
		for _, decorator := range n.Decorators {
			Walk(v, decorator)
		}
		for _, field := range n.Fields {
			Walk(v, field)
		}
		for _, method := range n.Methods {
			Walk(v, method)
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
