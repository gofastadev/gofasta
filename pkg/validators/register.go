package validators

import (
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"gorm.io/gorm"
)

// newBaseValidator creates a validator with common (framework-level) validations.
// Projects should call NewAppValidator() and then register additional validations
// on the returned AppValidator.Validate field.
func newBaseValidator(db *gorm.DB) (*validator.Validate, ut.Translator) {
	validate := validator.New()

	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)

	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validate, trans)

	// Common validators
	validate.RegisterValidation("uuid4_valid", isUUIDv4Valid)
	validate.RegisterValidation("is_record_deletable", isRecordDeletable(db))
	validate.RegisterValidation("is_record_exist_by_name_for_conflict", isRecordExistByName(db))
	validate.RegisterValidation("does_record_exist_by_id_for_verification", isRecordExistById(db))
	validate.RegisterValidation("is_valid_url", isValidURL)

	return validate, trans
}

// RegisterCommonValidators registers the framework's common validators on an existing validator instance.
func RegisterCommonValidators(validate *validator.Validate, db *gorm.DB) {
	validate.RegisterValidation("uuid4_valid", isUUIDv4Valid)
	validate.RegisterValidation("is_record_deletable", isRecordDeletable(db))
	validate.RegisterValidation("is_record_exist_by_name_for_conflict", isRecordExistByName(db))
	validate.RegisterValidation("does_record_exist_by_id_for_verification", isRecordExistById(db))
	validate.RegisterValidation("is_valid_url", isValidURL)
}

// RegisterTranslation is a helper for projects to register custom validation messages.
func RegisterTranslation(v *validator.Validate, trans ut.Translator, tag string, message string) {
	v.RegisterTranslation(tag, trans, func(ut ut.Translator) error {
		return ut.Add(tag, message, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field(), fe.Param())
		return t
	})
}
