package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractEmailFromToken(t *testing.T) {
	// Create a mock token payload
	payload := map[string]interface{}{
		"email": "test@example.com",
		"name":  "Test User",
	}
	
	// Encode the payload
	jsonPayload, _ := json.Marshal(payload)
	base64Payload := base64.StdEncoding.EncodeToString(jsonPayload)
	// Replace standard base64 chars with URL-safe chars
	base64Payload = strings.ReplaceAll(base64Payload, "+", "-")
	base64Payload = strings.ReplaceAll(base64Payload, "/", "_")
	// Remove padding
	base64Payload = strings.TrimRight(base64Payload, "=")
	
	// Create a mock token
	mockToken := "header." + base64Payload + ".signature"
	
	// Test valid token
	claims, err := ExtractEmailFromToken(mockToken)
	if err != nil {
		t.Fatalf("Failed to extract email from valid token: %v", err)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", claims.Email)
	}
	if claims.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got '%s'", claims.Name)
	}
	
	// Test invalid token format
	_, err = ExtractEmailFromToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token format, got nil")
	}
	
	// Test token without email
	payloadNoEmail := map[string]interface{}{
		"name": "Test User",
	}
	jsonPayloadNoEmail, _ := json.Marshal(payloadNoEmail)
	base64PayloadNoEmail := base64.StdEncoding.EncodeToString(jsonPayloadNoEmail)
	base64PayloadNoEmail = strings.ReplaceAll(base64PayloadNoEmail, "+", "-")
	base64PayloadNoEmail = strings.ReplaceAll(base64PayloadNoEmail, "/", "_")
	base64PayloadNoEmail = strings.TrimRight(base64PayloadNoEmail, "=")
	mockTokenNoEmail := "header." + base64PayloadNoEmail + ".signature"
	
	_, err = ExtractEmailFromToken(mockTokenNoEmail)
	if err == nil {
		t.Error("Expected error for token without email, got nil")
	}
}
