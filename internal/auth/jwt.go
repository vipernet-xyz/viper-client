package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Common JWT errors
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Config holds the JWT configuration
type Config struct {
	SecretKey     string
	TokenDuration time.Duration
}

// Service provides JWT authentication operations
type Service struct {
	config Config
}

// UserClaims represents the claims in a JWT token
type UserClaims struct {
	jwt.RegisteredClaims
	UserID         string `json:"user_id"`
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
}

// NewAuthService creates a new auth service
func NewAuthService(config Config) *Service {
	return &Service{
		config: config,
	}
}

// GenerateToken generates a new JWT token for a user
func (s *Service) GenerateToken(userID, providerUserID, email, name string) (string, error) {
	// Set token expiration time
	expirationTime := time.Now().Add(s.config.TokenDuration)

	// Create the JWT claims
	claims := &UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "viper-client",
			Subject:   userID,
		},
		UserID:         userID,
		ProviderUserID: providerUserID,
		Email:          email,
		Name:           name,
	}

	// Create a new token with the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*UserClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Extract the claims
	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// SimulateWeb3AuthToken simulates token verification from Web3Auth
// In a real application, this would validate a token from Web3Auth
func (s *Service) SimulateWeb3AuthToken(tokenString string) (*UserClaims, error) {
	// For simulation purposes, we just validate the JWT token
	return s.ValidateToken(tokenString)
}
