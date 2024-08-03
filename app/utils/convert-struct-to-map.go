package utils

import (
	"reflect"
	"regexp"
	"strings"
)

// Convert camelCase to snake_case
func camelToSnake(s string) string {
    // Insert underscores before capital letters and convert to lowercase
    re := regexp.MustCompile("([a-z0-9])([A-Z])")
    snake := re.ReplaceAllString(s, "${1}_${2}")
    return strings.ToLower(snake)
}

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
			snakeName := camelToSnake(fieldName)
			result[snakeName] = field.Elem().String()
		}
    }
    return result
}
