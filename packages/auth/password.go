package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"github.com/healtronlabs/gofasta/packages/core"
)

// PasswordService provides password hashing and verification
type PasswordService struct {
	config *PasswordConfig
}

// PasswordConfig represents password hashing configuration
type PasswordConfig struct {
	Memory      uint32 // Memory usage in KB
	Iterations  uint32 // Number of iterations
	Parallelism uint8  // Number of threads
	SaltLength  uint32 // Salt length in bytes
	KeyLength   uint32 // Key length in bytes
}

// DefaultPasswordConfig returns default password configuration
func DefaultPasswordConfig() *PasswordConfig {
	return &PasswordConfig{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// NewPasswordService creates a new password service
func NewPasswordService(config *PasswordConfig) *PasswordService {
	if config == nil {
		config = DefaultPasswordConfig()
	}
	return &PasswordService{
		config: config,
	}
}

// HashPassword hashes a password using Argon2id
func (s *PasswordService) HashPassword(password string) (string, error) {
	// Generate salt
	salt, err := s.generateSalt()
	if err != nil {
		return "", core.NewInternalServerException("Failed to generate salt", err)
	}

	// Hash password
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.config.Iterations,
		s.config.Memory,
		s.config.Parallelism,
		s.config.KeyLength,
	)

	// Encode to string
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		s.config.Memory,
		s.config.Iterations,
		s.config.Parallelism,
		encodedSalt,
		encodedHash,
	), nil
}

// VerifyPassword verifies a password against its hash
func (s *PasswordService) VerifyPassword(password, hashedPassword string) (bool, error) {
	// Parse the hash
	params, salt, hash, err := s.parseHash(hashedPassword)
	if err != nil {
		return false, core.NewBadRequestException("Invalid password hash format")
	}

	// Hash the input password with the same parameters
	inputHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		uint32(len(hash)),
	)

	// Compare hashes using constant-time comparison
	return subtle.ConstantTimeCompare(hash, inputHash) == 1, nil
}

// generateSalt generates a random salt
func (s *PasswordService) generateSalt() ([]byte, error) {
	salt := make([]byte, s.config.SaltLength)
	_, err := rand.Read(salt)
	return salt, err
}

// parseHash parses an Argon2id hash string
func (s *PasswordService) parseHash(hashedPassword string) (*PasswordConfig, []byte, []byte, error) {
	parts := strings.Split(hashedPassword, "$")
	if len(parts) != 6 {
		return nil, nil, nil, fmt.Errorf("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("unsupported hash algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid version")
	}

	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("unsupported version")
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid salt encoding")
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid hash encoding")
	}

	config := &PasswordConfig{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
	}

	return config, salt, hash, nil
}

// GenerateRandomPassword generates a random password
func (s *PasswordService) GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)
	
	for i := range password {
		randomBytes := make([]byte, 1)
		if _, err := rand.Read(randomBytes); err != nil {
			return "", core.NewInternalServerException("Failed to generate random password", err)
		}
		password[i] = charset[randomBytes[0]%byte(len(charset))]
	}

	return string(password), nil
}

// ValidatePasswordStrength validates password strength
func (s *PasswordService) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return core.NewBadRequestException("Password must be at least 8 characters long")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return core.NewBadRequestException("Password must contain at least one uppercase letter")
	}
	if !hasLower {
		return core.NewBadRequestException("Password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return core.NewBadRequestException("Password must contain at least one digit")
	}
	if !hasSpecial {
		return core.NewBadRequestException("Password must contain at least one special character")
	}

	return nil
}

// PasswordResetService handles password reset functionality
type PasswordResetService struct {
	passwordService *PasswordService
	jwtService      *JWTService
}

// NewPasswordResetService creates a new password reset service
func NewPasswordResetService(passwordService *PasswordService, jwtService *JWTService) *PasswordResetService {
	return &PasswordResetService{
		passwordService: passwordService,
		jwtService:      jwtService,
	}
}

// GenerateResetToken generates a password reset token
func (s *PasswordResetService) GenerateResetToken(userID, email string) (string, error) {
	// Generate a short-lived token for password reset (15 minutes)
	tokenPair, err := s.jwtService.GenerateTokenPair(userID, email, []string{"password_reset"})
	if err != nil {
		return "", err
	}
	return tokenPair.AccessToken, nil
}

// ValidateResetToken validates a password reset token
func (s *PasswordResetService) ValidateResetToken(token string) (*UserInfo, error) {
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return nil, core.NewUnauthorizedException("Invalid or expired reset token")
	}

	// Check if token has password_reset role
	hasResetRole := false
	for _, role := range claims.Roles {
		if role == "password_reset" {
			hasResetRole = true
			break
		}
	}

	if !hasResetRole {
		return nil, core.NewUnauthorizedException("Invalid reset token")
	}

	return &UserInfo{
		ID:    claims.UserID,
		Email: claims.Email,
	}, nil
}

// ResetPassword resets a user's password using a reset token
func (s *PasswordResetService) ResetPassword(token, newPassword string) error {
	// Validate token
	_, err := s.ValidateResetToken(token)
	if err != nil {
		return err
	}

	// Validate password strength
	if err := s.passwordService.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	// Hash new password
	_, err = s.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// In a real implementation, you would update the user's password in the database
	// This would typically be done through a user repository
	
	return nil
}