package utils

import (
	"regexp"
	"strings"

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
