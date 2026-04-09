package utils

import (
	"crypto/rand"
	"io"
	"math/big"
)

var cryptoRandReader io.Reader = rand.Reader

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
	n, err := rand.Int(cryptoRandReader, max)
	if err != nil {
		return 0, err
	}
	return CHARSET[n.Int64()], nil
}
