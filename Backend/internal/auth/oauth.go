package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// OAuthUser represents a user from OAuth providers
type OAuthUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Avatar   string `json:"avatar_url,omitempty"`
}

// OAuthHandlers handles OAuth-related endpoints
type OAuthHandlers struct {
	authService *AuthService
	jwtManager  *JWTManager
}

// NewOAuthHandlers creates a new OAuth handlers instance
func NewOAuthHandlers(authService *AuthService, jwtManager *JWTManager) *OAuthHandlers {
	return &OAuthHandlers{
		authService: authService,
		jwtManager:  jwtManager,
	}
}

// HandleOAuthCallback handles OAuth callback and creates/updates user
func (h *OAuthHandlers) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	var oauthUser OAuthUser
	if err := json.NewDecoder(r.Body).Decode(&oauthUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if user exists by email
	existingUser, err := h.authService.GetUserByEmail(r.Context(), oauthUser.Email)
	if err != nil && err.Error() != "user not found" {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var user *User
	if existingUser != nil {
		// Update existing user with OAuth info
		existingUser.OAuthProvider = oauthUser.Provider
		existingUser.OAuthID = oauthUser.ID
		existingUser.UpdatedAt = time.Now()
		
		if err := h.authService.UpdateUser(r.Context(), existingUser); err != nil {
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}
		user = existingUser
	} else {
		// Create new user from OAuth data
		user = &User{
			ID:            uuid.New().String(),
			Username:      generateUsernameFromEmail(oauthUser.Email),
			Email:         oauthUser.Email,
			OAuthProvider: oauthUser.Provider,
			OAuthID:       oauthUser.ID,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := h.authService.CreateUser(r.Context(), user); err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	// Generate JWT token
	token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Return user data and token
	response := map[string]interface{}{
		"success": true,
		"message": "OAuth authentication successful",
		"data": map[string]interface{}{
			"token": token,
			"user": map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
				"email":    user.Email,
				"provider": user.OAuthProvider,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateUsernameFromEmail creates a username from email
func generateUsernameFromEmail(email string) string {
	// Simple implementation: use part before @ symbol
	for i, char := range email {
		if char == '@' {
			return email[:i]
		}
	}
	return email
}