package codegen

// QueryParameterOptions holds options for query parameter extraction
type QueryParameterOptions struct {
	DefaultValue string
	Required     bool
	Type         string // "string", "int", "bool", "array", "float"
	Separator    string // for array types, default ","
	Transform    string // "lowercase", "uppercase", "trim"
}

// HeaderParameterOptions holds options for header parameter extraction
type HeaderParameterOptions struct {
	DefaultValue string
	Required     bool
	Type         string // "string", "int", "bool", "array", "float"
	Separator    string // for array types, default ","
	Transform    string // "lowercase", "uppercase", "trim"
	CaseInsensitive bool // whether header matching should be case insensitive (default: true)
}

// ParamConstraint represents a parameter constraint
type ParamConstraint struct {
	Type   string // "int", "guid", "regex", "min", "max", "range", "length", "minlength", "maxlength", "alpha", "bool"
	Value  string // constraint value (e.g., regex pattern, min/max values)
	Value2 string // second value for range constraints
}

// ParameterConstraintOptions holds options for parameter constraint validation
type ParameterConstraintOptions struct {
	Constraints []ParamConstraint
	Required    bool
	Transform   string // "lowercase", "uppercase", "trim"
}

// FieldInjectionConfig holds configuration for dependency injection
type FieldInjectionConfig struct {
	Token    string
	Optional bool
	Scope    string
}

// RouteInfo holds HTTP route information
type RouteInfo struct {
	Method string
	Path   string
}

// ModuleConfig holds module configuration information
type ModuleConfig struct {
	Controllers []string
	Providers   []string
	Imports     []string
	Exports     []string
}

// CatchFilterConfig holds configuration for error handling filters
type CatchFilterConfig struct {
	ErrorTypes []string // The error types this filter catches
	Scope      string   // "method", "controller", or "global"
	Handler    string   // The handler method name
}

// ValidationStructInfo holds information about a struct that needs validation
type ValidationStructInfo struct {
	Name   string
	Fields []*ValidationFieldInfo
}

// ValidationFieldInfo holds information about a field that needs validation
type ValidationFieldInfo struct {
	Name       string
	Type       string
	Tag        string
	Validators []ValidationRule
}

// ValidationRule represents a validation rule for a field
type ValidationRule struct {
	Type    string
	Args    []interface{}
	Message string
	Code    string
}