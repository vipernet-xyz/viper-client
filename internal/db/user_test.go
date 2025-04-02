package db

import (
	"os"
	"testing"
)

func TestUserRepository(t *testing.T) {
	// Skip if no DATABASE_URL environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	// Connect to the database
	db, err := New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = db.MigrateDB("")
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test create user
	user, err := db.CreateUser("test-provider-id", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be non-zero")
	}
	if user.ProviderUserID != "test-provider-id" {
		t.Errorf("Expected provider user ID to be 'test-provider-id', got '%s'", user.ProviderUserID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email to be 'test@example.com', got '%s'", user.Email)
	}
	if user.Name != "Test User" {
		t.Errorf("Expected name to be 'Test User', got '%s'", user.Name)
	}

	// Test get user by ID
	fetchedUser, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if fetchedUser.ID != user.ID {
		t.Errorf("Expected user ID to be %d, got %d", user.ID, fetchedUser.ID)
	}
	if fetchedUser.ProviderUserID != user.ProviderUserID {
		t.Errorf("Expected provider user ID to be '%s', got '%s'", user.ProviderUserID, fetchedUser.ProviderUserID)
	}

	// Test get user by provider ID
	fetchedUser, err = db.GetUserByProviderID("test-provider-id")
	if err != nil {
		t.Fatalf("Failed to get user by provider ID: %v", err)
	}

	if fetchedUser.ID != user.ID {
		t.Errorf("Expected user ID to be %d, got %d", user.ID, fetchedUser.ID)
	}
	if fetchedUser.Email != user.Email {
		t.Errorf("Expected email to be '%s', got '%s'", user.Email, fetchedUser.Email)
	}

	// Test update user
	updatedUser, err := db.UpdateUser(user.ID, "updated@example.com", "Updated User")
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	if updatedUser.ID != user.ID {
		t.Errorf("Expected user ID to be %d, got %d", user.ID, updatedUser.ID)
	}
	if updatedUser.Email != "updated@example.com" {
		t.Errorf("Expected email to be 'updated@example.com', got '%s'", updatedUser.Email)
	}
	if updatedUser.Name != "Updated User" {
		t.Errorf("Expected name to be 'Updated User', got '%s'", updatedUser.Name)
	}

	// Test get user by non-existent ID
	_, err = db.GetUserByID(9999)
	if err == nil {
		t.Error("Expected error when getting non-existent user by ID")
	}
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	// Test get user by non-existent provider ID
	_, err = db.GetUserByProviderID("non-existent-provider-id")
	if err == nil {
		t.Error("Expected error when getting non-existent user by provider ID")
	}
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	// Clean up
	_, err = db.DB.Exec("DELETE FROM users WHERE id = $1", user.ID)
	if err != nil {
		t.Fatalf("Failed to clean up test user: %v", err)
	}
}
