package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"echofs/pkg/auth"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserRepository struct {
	db *PostgresDB
}

func NewUserRepository(db *PostgresDB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(255) PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	
	-- Add role column if it doesn't exist (for existing databases)
	DO $$ 
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='users' AND column_name='role') THEN
			ALTER TABLE users ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'user';
		END IF;
	END $$;
	
	-- Update admin user if exists
	UPDATE users SET role = 'admin' WHERE username = 'admin';
	`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *UserRepository) CreateUser(ctx context.Context, username, email, password string) (*auth.User, error) {
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Check if this is the admin user
	role := "user"
	if username == "admin" {
		role = "admin"
	}

	user := &auth.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.ExecContext(ctx, query, user.ID, user.Username, user.Email, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" ||
			err.Error() == "pq: duplicate key value violates unique constraint \"users_username_key\"" {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	user := &auth.User{}
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*auth.User, error) {
	user := &auth.User{}
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) AuthenticateUser(ctx context.Context, email, password string) (*auth.User, error) {
	user, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
