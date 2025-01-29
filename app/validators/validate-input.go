package validators

import (
	"github.com/go-playground/validator/v10"
	"github.com/healtronlabs/gofasta/app/dtos"
	"github.com/healtronlabs/gofasta/app/utils"
	"gorm.io/gorm"
)

// ValidateInput validates the input struct and returns any validation errors
func ValidateInput(input interface{}, db *gorm.DB) []*dtos.TCommonAPIErrorDto {
	validate, trans, err := newValidator(db)
	if err != nil {
		return []*dtos.TCommonAPIErrorDto{{Message: "Validation initialization error"}}
	}

	valErrs := []*dtos.TCommonAPIErrorDto{}
	validationError := validate.Struct(input)
	if validationError != nil {
		if validationErrors, ok := validationError.(validator.ValidationErrors); ok {
			for _, vErr := range validationErrors {
				fieldName := utils.ConvertUpperCamelToLowerCamel(vErr.Field())
				valErrs = append(valErrs, &dtos.TCommonAPIErrorDto{Message: vErr.Translate(trans), FieldName: &fieldName})
			}
		}
	}

	return valErrs
}
