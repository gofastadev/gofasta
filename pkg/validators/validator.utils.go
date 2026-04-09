package validators

import (
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func getValue(fl validator.FieldLevel) string {
	if fl.Field().Type() == reflect.TypeOf(uuid.UUID{}) {
		return fl.Field().Interface().(uuid.UUID).String()
	}
	return fl.Field().String()
}
