package utils

import (
	"reflect"
)

func ConvertStructToMap(obj interface{}) map[string]string {
    result := make(map[string]string)
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
		if field.IsNil() {
			continue
		} else {
			fieldName := typeOfObj.Field(i).Name
			snakeName := CamelToSnake(fieldName)
			result[snakeName] = field.Elem().String()
		}
    }
    return result
}
