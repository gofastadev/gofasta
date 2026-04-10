package utils

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// CamelToSnake converts a camelCase or PascalCase identifier to snake_case.
func CamelToSnake(s string) string {
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}

// ParseIDStringIsValidUUID parses a string as a UUID and returns it alongside any parse error.
func ParseIDStringIsValidUUID(u string) (uuid.UUID, error) {
	value, err := uuid.Parse(u)
	return value, err
}

// ConvertUpperCamelToLowerCamel converts a PascalCase identifier to camelCase.
func ConvertUpperCamelToLowerCamel(input string) string {
	if input == "" {
		return input
	}
	var words []string
	start := 0
	for i, r := range input {
		if i != 0 && unicode.IsUpper(r) {
			words = append(words, input[start:i])
			start = i
		}
	}
	words = append(words, input[start:])
	words[0] = strings.ToLower(words[0])
	res := strings.Join(words, "")
	if len(res) == 2 && res == "iD" {
		res = "id"
	}
	return res
}
