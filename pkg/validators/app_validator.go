package validators

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/gofastadev/gofasta/pkg/types"
	"github.com/gofastadev/gofasta/pkg/utils"
	"gorm.io/gorm"
)

// AppValidator is a reusable validator instance created once at startup.
type AppValidator struct {
	Validate *validator.Validate
	Trans    ut.Translator
}

// NewAppValidator creates a validator with common validations registered.
// Projects can extend it by registering additional validations on Validate.
func NewAppValidator(db *gorm.DB) *AppValidator {
	validate, trans := newBaseValidator(db)
	return &AppValidator{
		Validate: validate,
		Trans:    trans,
	}
}

// ValidateStruct validates the input struct and returns any validation errors.
func (v *AppValidator) ValidateStruct(input interface{}) []*types.TCommonAPIErrorDto {
	valErrs := []*types.TCommonAPIErrorDto{}
	validationError := v.Validate.Struct(input)
	if validationError != nil {
		if validationErrors, ok := validationError.(validator.ValidationErrors); ok {
			for _, vErr := range validationErrors {
				fieldName := utils.ConvertUpperCamelToLowerCamel(vErr.Field())
				valErrs = append(valErrs, &types.TCommonAPIErrorDto{Message: vErr.Translate(v.Trans), FieldName: &fieldName})
			}
		}
	}
	return valErrs
}
