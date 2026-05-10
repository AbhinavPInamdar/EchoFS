package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthMiddleware_NoHeader(t *testing.T) {
	manager := NewJWTManager("secret", 1*time.Hour)
	mw := NewAuthMiddleware(manager)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth header")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	manager := NewJWTManager("secret", 1*time.Hour)
	mw := NewAuthMiddleware(manager)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid format")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "NotBearer token123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	manager := NewJWTManager("secret", 1*time.Hour)
	mw := NewAuthMiddleware(manager)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	manager := NewJWTManager("secret", 1*time.Hour)
	mw := NewAuthMiddleware(manager)

	token, _ := manager.GenerateToken("user-42", "test@test.com", "tester", "admin")

	var capturedClaims *Claims
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Error("expected claims in context")
			return
		}
		capturedClaims = claims
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedClaims == nil {
		t.Fatal("claims not captured")
	}
	if capturedClaims.UserID != "user-42" {
		t.Errorf("expected UserID 'user-42', got '%s'", capturedClaims.UserID)
	}
	if capturedClaims.Role != "admin" {
		t.Errorf("expected Role 'admin', got '%s'", capturedClaims.Role)
	}
}

func TestGetUserFromContext_NoUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	_, ok := GetUserFromContext(req.Context())
	if ok {
		t.Error("expected false when no user in context")
	}
}
