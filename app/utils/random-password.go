package utils

import (
	"crypto/rand"
	"math/big"
)

const CHARSET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?/~`"

func GeneratePassword(length int) (string, error) {
	password := make([]byte, length)
	for i := range password {
		char, err := randomChar()
		if err != nil {
			return "", err
		}
		password[i] = char
	}
	return string(password), nil
}

func randomChar() (byte, error) {
	max := big.NewInt(int64(len(CHARSET)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return CHARSET[n.Int64()], nil
}
