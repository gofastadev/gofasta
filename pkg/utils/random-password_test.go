package utils

import (
	"errors"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GeneratePassword ---

func TestGeneratePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
		{"length 1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd, err := GeneratePassword(tt.length)
			require.NoError(t, err)
			assert.Len(t, pwd, tt.length)
		})
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	// Two generated passwords should almost certainly differ
	pwd1, err := GeneratePassword(32)
	require.NoError(t, err)
	pwd2, err := GeneratePassword(32)
	require.NoError(t, err)
	assert.NotEqual(t, pwd1, pwd2)
}

func TestGeneratePassword_UsesCharset(t *testing.T) {
	pwd, err := GeneratePassword(100)
	require.NoError(t, err)
	for _, c := range pwd {
		assert.Contains(t, Charset, string(c), "password contains character not in Charset: %c", c)
	}
}

func TestGeneratePassword_RandError(t *testing.T) {
	oldReader := cryptoRandReader
	cryptoRandReader = iotest.ErrReader(errors.New("entropy failure"))
	defer func() { cryptoRandReader = oldReader }()

	_, err := GeneratePassword(8)
	require.Error(t, err)
}

func TestRandomChar_Success(t *testing.T) {
	c, err := randomChar()
	require.NoError(t, err)
	assert.Contains(t, Charset, string(c))
}
