package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/healtronlabs/gofasta/packages/core"
)

// ValidationPipe implements the Pipe interface for request validation
type ValidationPipe struct {
	validator  *validator.Validate
	translator ut.Translator
}

// NewValidationPipe creates a new validation pipe
func NewValidationPipe() *ValidationPipe {
	validate := validator.New()
	
	// Setup translator
	english := en.New()
	uni := ut.New(english, english)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validate, trans)

	// Register custom field name tag
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	pipe := &ValidationPipe{
		validator:  validate,
		translator: trans,
	}

	// Register custom validators
	pipe.registerCustomValidators()

	return pipe
}

// Transform implements the Pipe interface
func (p *ValidationPipe) Transform(value interface{}, metadata *core.PipeMetadata) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	// Validate the value
	if err := p.validator.Struct(value); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return nil, core.NewValidationException(p.formatValidationErrors(validationErrors))
		}
		return nil, core.NewBadRequestException("Validation failed")
	}

	return value, nil
}

// formatValidationErrors formats validation errors for user-friendly display
func (p *ValidationPipe) formatValidationErrors(errors validator.ValidationErrors) map[string][]string {
	errorMap := make(map[string][]string)

	for _, err := range errors {
		field := err.Field()
		message := err.Translate(p.translator)
		
		if errorMap[field] == nil {
			errorMap[field] = make([]string, 0)
		}
		errorMap[field] = append(errorMap[field], message)
	}

	return errorMap
}

// registerCustomValidators registers custom validation rules
func (p *ValidationPipe) registerCustomValidators() {
	// Phone number validator
	p.validator.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		phone := fl.Field().String()
		// Simple phone validation - would be more sophisticated in real implementation
		return len(phone) >= 10 && len(phone) <= 15
	})

	// Strong password validator
	p.validator.RegisterValidation("strong_password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		return len(password) >= 8 && 
			   strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") &&
			   strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") &&
			   strings.ContainsAny(password, "0123456789") &&
			   strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?")
	})

	// UUID validator
	p.validator.RegisterValidation("uuid", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		// Simple UUID validation - would use proper UUID library in real implementation
		return len(value) == 36 && strings.Count(value, "-") == 4
	})

	// Custom translations
	p.validator.RegisterTranslation("phone", p.translator, func(ut ut.Translator) error {
		return ut.Add("phone", "{0} must be a valid phone number", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("phone", fe.Field())
		return t
	})

	p.validator.RegisterTranslation("strong_password", p.translator, func(ut ut.Translator) error {
		return ut.Add("strong_password", "{0} must contain at least 8 characters including uppercase, lowercase, numbers and special characters", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("strong_password", fe.Field())
		return t
	})

	p.validator.RegisterTranslation("uuid", p.translator, func(ut ut.Translator) error {
		return ut.Add("uuid", "{0} must be a valid UUID", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("uuid", fe.Field())
		return t
	})
}

// ValidateStruct validates a struct and returns formatted errors
func (p *ValidationPipe) ValidateStruct(s interface{}) error {
	if err := p.validator.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return core.NewValidationException(p.formatValidationErrors(validationErrors))
		}
		return core.NewBadRequestException("Validation failed")
	}
	return nil
}

// ValidateField validates a single field
func (p *ValidationPipe) ValidateField(field interface{}, tag string) error {
	if err := p.validator.Var(field, tag); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return core.NewValidationException(p.formatValidationErrors(validationErrors))
		}
		return core.NewBadRequestException("Field validation failed")
	}
	return nil
}

// DTO represents a Data Transfer Object with validation
type DTO interface {
	Validate() error
}

// BaseDTO provides common validation functionality
type BaseDTO struct {
	validator *ValidationPipe
}

// NewBaseDTO creates a new BaseDTO
func NewBaseDTO() *BaseDTO {
	return &BaseDTO{
		validator: NewValidationPipe(),
	}
}

// Validate validates the DTO
func (dto *BaseDTO) Validate() error {
	return dto.validator.ValidateStruct(dto)
}

// Validation decorators for struct tags
type ValidationRule struct {
	Tag     string
	Message string
}

// Common validation tags
var (
	Required = ValidationRule{Tag: "required", Message: "This field is required"}
	Email    = ValidationRule{Tag: "email", Message: "Must be a valid email address"}
	Min      = func(min int) ValidationRule {
		return ValidationRule{Tag: "min=" + string(rune(min)), Message: "Must be at least " + string(rune(min)) + " characters"}
	}
	Max = func(max int) ValidationRule {
		return ValidationRule{Tag: "max=" + string(rune(max)), Message: "Must be at most " + string(rune(max)) + " characters"}
	}
	Length = func(min, max int) ValidationRule {
		return ValidationRule{Tag: "min=" + string(rune(min)) + ",max=" + string(rune(max)), Message: "Must be between " + string(rune(min)) + " and " + string(rune(max)) + " characters"}
	}
)

// ParseValidation parses validation rules from struct field
func ParseValidation(field reflect.StructField) []ValidationRule {
	rules := make([]ValidationRule, 0)
	
	if validateTag, ok := field.Tag.Lookup("validate"); ok {
		parts := strings.Split(validateTag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			rule := ValidationRule{Tag: part}
			
			// Add custom messages based on tag
			switch {
			case part == "required":
				rule.Message = "This field is required"
			case part == "email":
				rule.Message = "Must be a valid email address"
			case strings.HasPrefix(part, "min="):
				rule.Message = "Must be at least " + strings.TrimPrefix(part, "min=") + " characters"
			case strings.HasPrefix(part, "max="):
				rule.Message = "Must be at most " + strings.TrimPrefix(part, "max=") + " characters"
			}
			
			rules = append(rules, rule)
		}
	}
	
	return rules
}