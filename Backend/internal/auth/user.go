package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID            string    `json:"id" dynamodb:"id"`
	Username      string    `json:"username" dynamodb:"username"`
	Email         string    `json:"email" dynamodb:"email"`
	Password      string    `json:"-" dynamodb:"password"` // Never return password in JSON
	OAuthProvider string    `json:"oauth_provider,omitempty" dynamodb:"oauth_provider"`
	OAuthID       string    `json:"oauth_id,omitempty" dynamodb:"oauth_id"`
	CreatedAt     time.Time `json:"created_at" dynamodb:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" dynamodb:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func (r *RegisterRequest) Validate() error {
	if r.Username == "" {
		return NewValidationError("username is required")
	}
	if r.Email == "" {
		return NewValidationError("email is required")
	}
	if r.Password == "" {
		return NewValidationError("password is required")
	}
	if len(r.Password) < 6 {
		return NewValidationError("password must be at least 6 characters")
	}
	return nil
}

func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return NewValidationError("email is required")
	}
	if r.Password == "" {
		return NewValidationError("password is required")
	}
	return nil
}

func GenerateUserID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func NewUser(req RegisterRequest) (*User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        GenerateUserID(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}