package auth

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	// Setup test configuration
	config := Config{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
	}
	authService := NewAuthService(config)

	// Test user data
	userID := "1"
	providerUserID := "web3auth-123456"
	email := "test@example.com"
	name := "Test User"

	// Generate a token
	token, err := authService.GenerateToken(userID, providerUserID, email, name)
	if err != nil {
		t.Fatalf("Error generating token: %v", err)
	}

	if token == "" {
		t.Fatal("Generated token is empty")
	}

	// Validate the token
	claims, err := authService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Error validating token: %v", err)
	}

	// Verify claims
	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.ProviderUserID != providerUserID {
		t.Errorf("Expected ProviderUserID %s, got %s", providerUserID, claims.ProviderUserID)
	}
	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}
	if claims.Name != name {
		t.Errorf("Expected Name %s, got %s", name, claims.Name)
	}
}

func TestInvalidToken(t *testing.T) {
	// Setup test configuration
	config := Config{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
	}
	authService := NewAuthService(config)

	// Test with invalid token
	invalidToken := "invalid.token.string"
	_, err := authService.ValidateToken(invalidToken)
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}
}

func TestExpiredToken(t *testing.T) {
	// Setup test configuration with very short duration
	config := Config{
		SecretKey:     "test-secret-key",
		TokenDuration: 1 * time.Millisecond, // Very short duration to force expiration
	}
	authService := NewAuthService(config)

	// Generate a token that will expire immediately
	token, err := authService.GenerateToken("1", "web3auth-123456", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Error generating token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate the expired token
	_, err = authService.ValidateToken(token)
	if err != ErrExpiredToken {
		t.Fatalf("Expected ErrExpiredToken, got %v", err)
	}
}

func TestSimulateWeb3AuthToken(t *testing.T) {
	// Setup test configuration
	config := Config{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
	}
	authService := NewAuthService(config)

	// Test user data
	userID := "1"
	providerUserID := "web3auth-123456"
	email := "test@example.com"
	name := "Test User"

	// Generate a token
	token, err := authService.GenerateToken(userID, providerUserID, email, name)
	if err != nil {
		t.Fatalf("Error generating token: %v", err)
	}

	// Simulate Web3Auth token verification (in our case, it's just JWT validation)
	claims, err := authService.SimulateWeb3AuthToken(token)
	if err != nil {
		t.Fatalf("Error validating Web3Auth token: %v", err)
	}

	// Verify claims
	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.ProviderUserID != providerUserID {
		t.Errorf("Expected ProviderUserID %s, got %s", providerUserID, claims.ProviderUserID)
	}
}
