package auth

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	manager := NewJWTManager("test-secret-key", 1*time.Hour)

	token, err := manager.GenerateToken("user-123", "test@example.com", "testuser", "user")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got '%s'", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got '%s'", claims.Email)
	}
	if claims.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got '%s'", claims.Username)
	}
	if claims.Role != "user" {
		t.Errorf("expected Role 'user', got '%s'", claims.Role)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	// Token that expires immediately
	manager := NewJWTManager("test-secret", -1*time.Second)

	token, err := manager.GenerateToken("user-1", "a@b.com", "user", "user")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestJWTManager_InvalidToken(t *testing.T) {
	manager := NewJWTManager("secret", 1*time.Hour)

	_, err := manager.ValidateToken("not.a.valid.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestJWTManager_WrongSecret(t *testing.T) {
	manager1 := NewJWTManager("secret-1", 1*time.Hour)
	manager2 := NewJWTManager("secret-2", 1*time.Hour)

	token, _ := manager1.GenerateToken("user-1", "a@b.com", "user", "user")

	_, err := manager2.ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating with wrong secret")
	}
}

func TestHashPassword_And_CheckPassword(t *testing.T) {
	password := "my-secure-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == password {
		t.Fatal("hash should not equal plaintext password")
	}

	// Correct password
	if err := CheckPassword(hash, password); err != nil {
		t.Errorf("CheckPassword should succeed for correct password: %v", err)
	}

	// Wrong password
	if err := CheckPassword(hash, "wrong-password"); err == nil {
		t.Error("CheckPassword should fail for wrong password")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	token1, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("GenerateSecureToken failed: %v", err)
	}
	if len(token1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64 char hex string, got %d chars", len(token1))
	}

	token2, _ := GenerateSecureToken(32)
	if token1 == token2 {
		t.Error("two generated tokens should not be equal")
	}
}
