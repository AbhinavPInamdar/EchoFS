package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"echofs/pkg/auth"
	"echofs/pkg/database"
)

type AuthHandler struct {
	userRepo   *database.UserRepository
	jwtManager *auth.JWTManager
	logger     *log.Logger
}

func NewAuthHandler(userRepo *database.UserRepository, jwtManager *auth.JWTManager, logger *log.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.sendErrorResponse(w, "Username, email, and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		h.sendErrorResponse(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.userRepo.CreateUser(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		if err == database.ErrUserAlreadyExists {
			h.sendErrorResponse(w, "User with this email or username already exists", http.StatusConflict)
			return
		}
		h.logger.Printf("Failed to create user: %v", err)
		h.sendErrorResponse(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, user.Username, user.Role)
	if err != nil {
		h.logger.Printf("Failed to generate token: %v", err)
		h.sendErrorResponse(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	response := auth.AuthResponse{
		Success: true,
		Message: "User registered successfully",
		Token:   token,
		User:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		h.sendErrorResponse(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.userRepo.AuthenticateUser(ctx, req.Email, req.Password)
	if err != nil {
		if err == database.ErrInvalidCredentials {
			h.sendErrorResponse(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		h.logger.Printf("Failed to authenticate user: %v", err)
		h.sendErrorResponse(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, user.Username, user.Role)
	if err != nil {
		h.logger.Printf("Failed to generate token: %v", err)
		h.sendErrorResponse(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	response := auth.AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
		User:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		h.sendErrorResponse(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		h.logger.Printf("Failed to get user: %v", err)
		h.sendErrorResponse(w, "Failed to get user profile", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"user":    user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := auth.AuthResponse{
		Success: false,
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
