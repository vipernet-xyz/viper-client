package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Common errors
var (
	ErrInvalidToken = errors.New("invalid token")
)

// EmailClaims represents the claims when extracting from a base64 token
type EmailClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ExtractEmailFromToken extracts email from a token using base64 decoding
func ExtractEmailFromToken(tokenString string) (*EmailClaims, error) {
	// Split the token
	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 {
		return nil, ErrInvalidToken
	}

	// Get the payload part (second part)
	base64Url := parts[1]
	
	// Replace characters as needed
	base64Str := strings.ReplaceAll(base64Url, "-", "+")
	base64Str = strings.ReplaceAll(base64Str, "_", "/")
	
	// Add padding if needed
	switch len(base64Str) % 4 {
	case 2:
		base64Str += "=="
	case 3:
		base64Str += "="
	}
	
	// Decode the base64 string
	decoded, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	
	// Parse the JSON
	var claims EmailClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	// Validate email presence
	if claims.Email == "" {
		return nil, errors.New("email not found in token")
	}
	
	return &claims, nil
}
