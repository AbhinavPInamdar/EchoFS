package auth

import (
	"context"
	"log"
)

type AuthService struct {
	userRepo   *UserRepository
	jwtManager *JWTManager
	logger     *log.Logger
}

func NewAuthService(userRepo *UserRepository, jwtManager *JWTManager, logger *log.Logger) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	s.logger.Printf("Registering new user: %s", req.Email)

	user, err := NewUser(req)
	if err != nil {
		return nil, err
	}

	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		s.logger.Printf("Failed to create user %s: %v", req.Email, err)
		return nil, err
	}

	token, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		s.logger.Printf("Failed to generate token for user %s: %v", user.ID, err)
		return nil, err
	}

	s.logger.Printf("Successfully registered user: %s (ID: %s)", user.Email, user.ID)

	return &AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	s.logger.Printf("Login attempt for user: %s", req.Email)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Printf("User not found: %s", req.Email)
		return nil, ErrUserNotFound
	}

	if !CheckPassword(req.Password, user.Password) {
		s.logger.Printf("Invalid password for user: %s", req.Email)
		return nil, ErrInvalidPassword
	}

	token, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		s.logger.Printf("Failed to generate token for user %s: %v", user.ID, err)
		return nil, err
	}

	s.logger.Printf("Successfully logged in user: %s (ID: %s)", user.Email, user.ID)

	return &AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *AuthService) GetUserProfile(ctx context.Context, userID string) (*User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

func (s *AuthService) RefreshToken(ctx context.Context, tokenString string) (string, error) {
	return s.jwtManager.RefreshToken(tokenString)
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	return s.jwtManager.ValidateToken(tokenString)
}

func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetUserByEmail(ctx, email)
}

func (s *AuthService) CreateUser(ctx context.Context, user *User) error {
	return s.userRepo.CreateUser(ctx, user)
}

func (s *AuthService) UpdateUser(ctx context.Context, user *User) error {
	return s.userRepo.UpdateUser(ctx, user)
}