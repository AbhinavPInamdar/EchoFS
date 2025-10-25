package auth

import (
	"encoding/json"
	"net/http"
)

// SimpleOAuthHandler creates JWT tokens for OAuth users without database storage
type SimpleOAuthHandler struct {
	jwtManager *JWTManager
}

func NewSimpleOAuthHandler(jwtManager *JWTManager) *SimpleOAuthHandler {
	return &SimpleOAuthHandler{
		jwtManager: jwtManager,
	}
}

func (h *SimpleOAuthHandler) HandleSimpleOAuth(w http.ResponseWriter, r *http.Request) {
	var oauthUser OAuthUser
	if err := json.NewDecoder(r.Body).Decode(&oauthUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate JWT token directly without database storage
	token, err := h.jwtManager.GenerateToken(oauthUser.ID, oauthUser.Name, oauthUser.Email)
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
				"user_id":  oauthUser.ID,
				"username": oauthUser.Name,
				"email":    oauthUser.Email,
				"provider": oauthUser.Provider,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}