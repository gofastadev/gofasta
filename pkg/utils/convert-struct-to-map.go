package utils

import (
	"reflect"
)

// ConvertStructToMap reflects over obj and returns a snake_case-keyed map
// of its non-nil pointer, interface, map, slice, channel, and function fields.
// It is intended for building GORM update-only change sets.
func ConvertStructToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	typeOfObj := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		switch field.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
			if field.IsNil() {
				continue
			}
			fieldName := typeOfObj.Field(i).Name
			snakeName := CamelToSnake(fieldName)
			result[snakeName] = field.Interface()
		}
	}
	return result
}
