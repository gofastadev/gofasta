package encryption

import (
	"encoding/base64"
	"errors"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validKey = "01234567890123456789012345678901" // exactly 32 bytes

func TestNewEncrypter(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		expectErr bool
		errMsg    string
	}{
		{"valid 32-byte key", validKey, false, ""},
		{"too short key", "short", true, "encryption key must be exactly 32 bytes, got 5"},
		{"too long key", validKey + "extra", true, "encryption key must be exactly 32 bytes, got 37"},
		{"empty key", "", true, "encryption key must be exactly 32 bytes, got 0"},
		{"16-byte key", "0123456789012345", true, "encryption key must be exactly 32 bytes, got 16"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewEncrypter(tt.key)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, enc)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, enc)
			}
		})
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	enc, err := NewEncrypter(validKey)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "hello world"},
		{"empty string", ""},
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "こんにちは世界"},
		{"long text", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."},
		{"newlines", "line1\nline2\nline3"},
		{"json payload", `{"key": "value", "num": 42}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := enc.Encrypt(tt.plaintext)
			require.NoError(t, err)
			assert.NotEmpty(t, ciphertext)
			assert.NotEqual(t, tt.plaintext, ciphertext)

			// Verify it's valid base64
			_, err = base64.StdEncoding.DecodeString(ciphertext)
			require.NoError(t, err)

			decrypted, err := enc.Decrypt(ciphertext)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncrypt_ProducesDifferentCiphertexts(t *testing.T) {
	enc, err := NewEncrypter(validKey)
	require.NoError(t, err)

	ct1, err := enc.Encrypt("same plaintext")
	require.NoError(t, err)
	ct2, err := enc.Encrypt("same plaintext")
	require.NoError(t, err)

	// Due to random nonce, same plaintext should produce different ciphertexts
	assert.NotEqual(t, ct1, ct2)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	enc, err := NewEncrypter(validKey)
	require.NoError(t, err)

	_, err = enc.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	enc, err := NewEncrypter(validKey)
	require.NoError(t, err)

	ciphertext, err := enc.Encrypt("hello")
	require.NoError(t, err)

	// Decode, corrupt, re-encode
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	require.NoError(t, err)

	// Flip some bytes in the ciphertext portion (after nonce)
	if len(raw) > 13 {
		raw[13] ^= 0xFF
	}
	corrupted := base64.StdEncoding.EncodeToString(raw)

	_, err = enc.Decrypt(corrupted)
	assert.Error(t, err)
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	enc, err := NewEncrypter(validKey)
	require.NoError(t, err)

	// Encode something shorter than nonce size (12 bytes for GCM)
	short := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = enc.Decrypt(short)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecrypt_WrongKey(t *testing.T) {
	enc1, err := NewEncrypter(validKey)
	require.NoError(t, err)

	enc2, err := NewEncrypter("abcdefghijklmnopqrstuvwxyz012345") // different 32-byte key
	require.NoError(t, err)

	ciphertext, err := enc1.Encrypt("secret data")
	require.NoError(t, err)

	_, err = enc2.Decrypt(ciphertext)
	assert.Error(t, err)
}

func TestEncrypt_RandReaderError(t *testing.T) {
	enc, err := NewEncrypter(validKey)
	require.NoError(t, err)

	oldReader := randReader
	randReader = iotest.ErrReader(errors.New("entropy failure"))
	defer func() { randReader = oldReader }()

	_, err = enc.Encrypt("hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entropy failure")
}
