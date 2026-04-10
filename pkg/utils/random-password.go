package utils

import (
	"crypto/rand"
	"math/big"
)

var cryptoRandReader = rand.Reader

// Charset is the set of characters used by GeneratePassword.
const Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?/~`"

// GeneratePassword returns a random password of the given length drawn from Charset.
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
	upper := big.NewInt(int64(len(Charset)))
	n, err := rand.Int(cryptoRandReader, upper)
	if err != nil {
		return 0, err
	}
	return Charset[n.Int64()], nil
}
