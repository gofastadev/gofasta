package core

import (
	"testing"
)

// TestGetDecoratorTypeBasic tests the basic GetDecoratorType function
func TestGetDecoratorTypeBasic(t *testing.T) {
	tests := []struct {
		name         string
		decoratorName string
		expected     DecoratorType
	}{
		{"Controller", "Controller", ControllerDecorator},
		{"Injectable", "Injectable", InjectableDecorator},
		{"Get", "Get", GetDecorator},
		{"Post", "Post", PostDecorator},
		{"IsString", "IsString", IsStringDecorator},
		{"IsEmail", "IsEmail", IsEmailDecorator},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDecoratorType(tt.decoratorName)
			if result != tt.expected {
				t.Errorf("GetDecoratorType(%q) = %v, expected %v", tt.decoratorName, result, tt.expected)
			}
		})
	}
}

// TestIsRouteDecoratorBasic tests the basic IsRouteDecorator function
func TestIsRouteDecoratorBasic(t *testing.T) {
	tests := []struct {
		name         string
		decoratorType DecoratorType
		expected     bool
	}{
		{"GetDecorator", GetDecorator, true},
		{"PostDecorator", PostDecorator, true},
		{"PutDecorator", PutDecorator, true},
		{"DeleteDecorator", DeleteDecorator, true},
		{"ControllerDecorator", ControllerDecorator, false},
		{"ServiceDecorator", ServiceDecorator, false},
		{"IsEmailDecorator", IsEmailDecorator, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRouteDecorator(tt.decoratorType)
			if result != tt.expected {
				t.Errorf("IsRouteDecorator(%v) = %v, expected %v", tt.decoratorType, result, tt.expected)
			}
		})
	}
}

// TestIsValidationDecoratorBasic tests the basic IsValidationDecorator function
func TestIsValidationDecoratorBasic(t *testing.T) {
	tests := []struct {
		name         string
		decoratorType DecoratorType
		expected     bool
	}{
		{"IsStringDecorator", IsStringDecorator, true},
		{"IsEmailDecorator", IsEmailDecorator, true},
		{"IsOptionalDecorator", IsOptionalDecorator, true},
		{"MinDecorator", MinDecorator, true},
		{"MaxDecorator", MaxDecorator, true},
		{"ControllerDecorator", ControllerDecorator, false},
		{"GetDecorator", GetDecorator, false},
		{"BodyDecorator", BodyDecorator, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidationDecorator(tt.decoratorType)
			if result != tt.expected {
				t.Errorf("IsValidationDecorator(%v) = %v, expected %v", tt.decoratorType, result, tt.expected)
			}
		})
	}
}

// TestIsParameterDecoratorBasic tests the basic IsParameterDecorator function
func TestIsParameterDecoratorBasic(t *testing.T) {
	tests := []struct {
		name         string
		decoratorType DecoratorType
		expected     bool
	}{
		{"BodyDecorator", BodyDecorator, true},
		{"ParamDecorator", ParamDecorator, true},
		{"QueryDecorator", QueryDecorator, true},
		{"SessionDecorator", SessionDecorator, true},
		{"GetDecorator", GetDecorator, false},
		{"ControllerDecorator", ControllerDecorator, false},
		{"IsEmailDecorator", IsEmailDecorator, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsParameterDecorator(tt.decoratorType)
			if result != tt.expected {
				t.Errorf("IsParameterDecorator(%v) = %v, expected %v", tt.decoratorType, result, tt.expected)
			}
		})
	}
}

// TestIsClassDecoratorBasic tests the basic IsClassDecorator function
func TestIsClassDecoratorBasic(t *testing.T) {
	tests := []struct {
		name         string
		decoratorType DecoratorType
		expected     bool
	}{
		{"ControllerDecorator", ControllerDecorator, true},
		{"ServiceDecorator", ServiceDecorator, true},
		{"InjectableDecorator", InjectableDecorator, true},
		{"ModuleDecorator", ModuleDecorator, true},
		{"GetDecorator", GetDecorator, false},
		{"BodyDecorator", BodyDecorator, false},
		{"IsEmailDecorator", IsEmailDecorator, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsClassDecorator(tt.decoratorType)
			if result != tt.expected {
				t.Errorf("IsClassDecorator(%v) = %v, expected %v", tt.decoratorType, result, tt.expected)
			}
		})
	}
}