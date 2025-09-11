package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// RandomString generates a random string of specified length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// GenerateTransactionID generates a unique transaction ID
func GenerateTransactionID(username string) string {
	return fmt.Sprintf("%s_%d_%s", username, GetCurrentTimestamp(), RandomString(8))
}

// SanitizeString removes special characters and normalizes string
func SanitizeString(input string) string {
	// Remove leading/trailing spaces and convert to lowercase
	sanitized := strings.TrimSpace(strings.ToLower(input))

	// Replace multiple spaces with single space
	sanitized = strings.Join(strings.Fields(sanitized), " ")

	return sanitized
}

// ValidateUsername checks if username meets requirements
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}
	if len(username) > 50 {
		return fmt.Errorf("username must be at most 50 characters long")
	}

	// Check for valid characters (alphanumeric and underscore only)
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return fmt.Errorf("username contains invalid characters")
		}
	}

	return nil
}

// MaskSensitiveData masks sensitive information for logging
func MaskSensitiveData(data string) string {
	if len(data) <= 4 {
		return strings.Repeat("*", len(data))
	}

	return data[:2] + strings.Repeat("*", len(data)-4) + data[len(data)-2:]
}
