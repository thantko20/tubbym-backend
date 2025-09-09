package domain

import "time"

type AuthProvider string

const (
	AuthProviderPassword AuthProvider = "password"
	AuthProviderGoogle   AuthProvider = "google"
	AuthProviderGitHub   AuthProvider = "github"
)

const (
	ErrCodeAuthInvalidCredentials ErrorCode = 1001
	ErrCodeAuthUserNotFound       ErrorCode = 1002
	ErrCodeAuthInvalidProvider    ErrorCode = 1003
	ErrCodeAuthInvalidSession     ErrorCode = 1004
)

type Session struct {
	ID        string       `json:"id" db:"id"`
	UserID    string       `json:"userId" db:"user_id"`
	Token     string       `json:"token" db:"token"`
	Provider  AuthProvider `json:"provider" db:"provider"`
	ExpiredAt time.Time    `json:"expiredAt" db:"expired_at"`
	CreatedAt time.Time    `json:"createdAt" db:"created_at"`
	DeletedAt *time.Time   `json:"deletedAt" db:"deleted_at"`
}

type ValidateSessionDTO struct {
	Session Session
	User    User
}
