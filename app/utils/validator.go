package utils

import (
	"fmt"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	dtos "github.com/healtronlabs/go_gql_template/app/dtos"
)

func IsUUIDv4(fl validator.FieldLevel) bool {
	id := fl.Field().String()
	err := ValidateIdStringIsValidUUID(id)
	return err == nil
}

// NewValidator creates and returns a new validator instance with custom error messages
func newValidator() (*validator.Validate, ut.Translator, error) {
	validate := validator.New()

	// Create a universal translator and register English translations
	en := en.New()
	uni := ut.New(en, en)

	trans, found := uni.GetTranslator("en")
	if !found {
		return nil, nil, fmt.Errorf("translator not found")
	}

	// Register English translations
	err := en_translations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		return nil, nil, err
	}

	// Custom messages for different validation tags
	customMessages := map[string]string{
		"required": "is a required field",
		"email":    "must be a valid email address",
		"uuid4":    "must be a valid UUIDV4",
		"min":      "must be at least {1} characters long",
		"max":      "must be at most {1} characters long",
		"len":      "must be exactly {1} characters long",
		// Add more custom messages as needed
	}

	for tag, message := range customMessages {
		validate.RegisterTranslation(tag, trans, func(ut ut.Translator) error {
			return ut.Add(tag, message, true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T(tag, fe.Field(), fe.Param())
			return t
		})
	}
	return validate, trans, nil
}

func ValidateInput(input interface{}) []*dtos.TCommonAPIErrorDto {
	validate, trans, err := newValidator()
	if err != nil {
		return []*dtos.TCommonAPIErrorDto{{Message: "Validation initialization error"}}
	}

	valErrs := []*dtos.TCommonAPIErrorDto{}
	validationError := validate.Struct(input)
	if validationError != nil {
		if validationErrors, ok := validationError.(validator.ValidationErrors); ok {
			for _, vErr := range validationErrors {
				fieldName := ConvertUpperCamelToLowerCamel(vErr.Field())
				valErrs = append(valErrs, &dtos.TCommonAPIErrorDto{Message: vErr.Translate(trans), FieldName: &fieldName})
			}
		}
	}

	return valErrs
}
