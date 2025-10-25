package auth

import (
	"encoding/json"
	"net/http"
)

type AuthHandlers struct {
	authService        *AuthService
	OAuthHandlers      *OAuthHandlers
	SimpleOAuthHandler *SimpleOAuthHandler
}

func NewAuthHandlers(authService *AuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
	}
}

func (h *AuthHandlers) SetOAuthHandlers(oauthHandlers *OAuthHandlers) {
	h.OAuthHandlers = oauthHandlers
}

func (h *AuthHandlers) SetSimpleOAuthHandler(simpleOAuthHandler *SimpleOAuthHandler) {
	h.SimpleOAuthHandler = simpleOAuthHandler
}

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if IsValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if IsAuthError(err) {
			authErr := err.(*AuthError)
			http.Error(w, authErr.Message, authErr.Code)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User registered successfully",
		"data":    response,
	})
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response, err := h.authService.Login(r.Context(), req)
	if err != nil {
		if IsValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if IsAuthError(err) {
			authErr := err.(*AuthError)
			http.Error(w, authErr.Message, authErr.Code)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Login successful",
		"data":    response,
	})
}

func (h *AuthHandlers) Profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	profile, err := h.authService.GetUserProfile(r.Context(), user.UserID)
	if err != nil {
		if IsAuthError(err) {
			authErr := err.(*AuthError)
			http.Error(w, authErr.Message, authErr.Code)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    profile,
	})
}

func (h *AuthHandlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	newToken, err := h.authService.RefreshToken(r.Context(), req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"token": newToken,
		},
	})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For JWT, logout is handled client-side by removing the token
	// In a more sophisticated system, you might maintain a blacklist
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}