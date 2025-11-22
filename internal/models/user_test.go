package models

import (
	"testing"
	"time"
)

func TestUserModel(t *testing.T) {
	user := User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Bio:      "Test bio",
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
	}
}

func TestUserCreatedAt(t *testing.T) {
	now := time.Now()
	user := User{
		ID:        1,
		Username:  "testuser",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if user.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if user.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}
