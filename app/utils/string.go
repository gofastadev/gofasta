package utils

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

func CamelToSnake(s string) string {
    re := regexp.MustCompile("([a-z0-9])([A-Z])")
    snake := re.ReplaceAllString(s, "${1}_${2}")
    return strings.ToLower(snake)
}

func ValidateIdStringIsValidUUID(u string) (error) {
    _, err := uuid.Parse(u)
    return err
}

func ConvertUpperCamelToLowerCamel(input string) string {
	if len(input) == 0 {
		return input
	}

	// Split the string into words based on uppercase letters
	var words []string
	start := 0
	for i, r := range input {
		if i != 0 && unicode.IsUpper(r) {
			words = append(words, input[start:i])
			start = i
		}
	}
	words = append(words, input[start:])

	// Convert the first word to lowercase
	words[0] = strings.ToLower(words[0])

	// Join the words together
	return strings.Join(words, "")
}
