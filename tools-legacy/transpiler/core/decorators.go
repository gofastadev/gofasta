package core

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

	// WebSocket decorators - Gateway
	WebSocketGatewayDecorator
	
	// WebSocket decorators - Message handling
	SubscribeMessageDecorator
	OnMessageDecorator
	MessagePatternDecorator
	
	// WebSocket decorators - Lifecycle
	OnGatewayConnectionDecorator
	OnGatewayDisconnectDecorator
	OnGatewayInitDecorator
	
	// WebSocket decorators - Parameters
	MessageBodyDecorator
	ConnectedSocketDecorator
	MessageAckDecorator
	RoomsDecorator
	NamespaceDecorator
	CurrentUserDecorator
	ClientIPDecorator
	DisconnectReasonDecorator
	EventNameDecorator
	RawMessageDecorator
	ServerDecorator
	
	// WebSocket decorators - Client
	WebSocketClientDecorator
	WebSocketTestClientDecorator
	WebSocketIntegrationTestDecorator

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
	
	// WebSocket decorators
	"WebSocketGateway":         WebSocketGatewayDecorator,
	"SubscribeMessage":         SubscribeMessageDecorator,
	"OnMessage":               OnMessageDecorator,
	"MessagePattern":          MessagePatternDecorator,
	"OnGatewayConnection":     OnGatewayConnectionDecorator,
	"OnGatewayDisconnect":     OnGatewayDisconnectDecorator,
	"OnGatewayInit":           OnGatewayInitDecorator,
	"MessageBody":             MessageBodyDecorator,
	"ConnectedSocket":         ConnectedSocketDecorator,
	"MessageAck":              MessageAckDecorator,
	"Rooms":                   RoomsDecorator,
	"Namespace":               NamespaceDecorator,
	"CurrentUser":             CurrentUserDecorator,
	"ClientIP":                ClientIPDecorator,
	"DisconnectReason":        DisconnectReasonDecorator,
	"EventName":               EventNameDecorator,
	"RawMessage":              RawMessageDecorator,
	"Server":                  ServerDecorator,
	"WebSocketClient":         WebSocketClientDecorator,
	"WebSocketTestClient":     WebSocketTestClientDecorator,
	"WebSocketIntegrationTest": WebSocketIntegrationTestDecorator,
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

// IsWebSocketDecorator checks if a decorator type is a WebSocket decorator
func IsWebSocketDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= WebSocketGatewayDecorator && decoratorType <= WebSocketIntegrationTestDecorator
}

// IsWebSocketGatewayDecorator checks if a decorator type is a WebSocket gateway decorator
func IsWebSocketGatewayDecorator(decoratorType DecoratorType) bool {
	return decoratorType == WebSocketGatewayDecorator
}

// IsWebSocketMessageDecorator checks if a decorator type is a WebSocket message handling decorator
func IsWebSocketMessageDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= SubscribeMessageDecorator && decoratorType <= MessagePatternDecorator
}

// IsWebSocketLifecycleDecorator checks if a decorator type is a WebSocket lifecycle decorator
func IsWebSocketLifecycleDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= OnGatewayConnectionDecorator && decoratorType <= OnGatewayInitDecorator
}

// IsWebSocketParameterDecorator checks if a decorator type is a WebSocket parameter decorator
func IsWebSocketParameterDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= MessageBodyDecorator && decoratorType <= ServerDecorator
}

// IsWebSocketClientDecorator checks if a decorator type is a WebSocket client decorator
func IsWebSocketClientDecorator(decoratorType DecoratorType) bool {
	return decoratorType >= WebSocketClientDecorator && decoratorType <= WebSocketIntegrationTestDecorator
}