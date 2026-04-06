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

func ParseIdStringIsValidUUID(u string) (uuid.UUID, error) {
	value, err := uuid.Parse(u)
	return value, err
}

func ConvertUpperCamelToLowerCamel(input string) string {
	if len(input) == 0 {
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
