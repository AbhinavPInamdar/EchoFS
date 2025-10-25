package auth

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

type AuthError struct {
	Message string
	Code    int
}

func (e *AuthError) Error() string {
	return e.Message
}

func NewAuthError(message string, code int) *AuthError {
	return &AuthError{Message: message, Code: code}
}

var (
	ErrUserNotFound     = NewAuthError("user not found", 404)
	ErrInvalidPassword  = NewAuthError("invalid password", 401)
	ErrUserExists       = NewAuthError("user already exists", 409)
	ErrInvalidToken     = NewAuthError("invalid token", 401)
	ErrTokenExpired     = NewAuthError("token expired", 401)
	ErrUnauthorized     = NewAuthError("unauthorized", 401)
	ErrForbidden        = NewAuthError("forbidden", 403)
)

func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

func IsAuthError(err error) bool {
	_, ok := err.(*AuthError)
	return ok
}