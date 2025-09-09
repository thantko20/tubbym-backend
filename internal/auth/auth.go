package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/thantko20/tubbym-backend/internal/domain"
	"github.com/thantko20/tubbym-backend/internal/repository"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	sessionTokenBytes = 16
	sessionDuration   = 7 * 24 * time.Hour
)

var (
	envOnce sync.Once
)

type AuthService interface {
	LoginWithProvider(provider domain.AuthProvider) (string, error)
	HandleProviderCallback(ctx context.Context, provider domain.AuthProvider, code string) (*domain.Session, error)
	ValidateSession(token string) (*domain.ValidateSessionDTO, *domain.AppError)
	Logout(ctx context.Context, token string) *domain.AppError
}

type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	httpClient  *http.Client
}

// NewAuthService creates a new authentication service instance
func NewAuthService(db *sql.DB) AuthService {
	// Load environment variables only once
	envOnce.Do(func() {
		if err := godotenv.Load(); err != nil {
			slog.Warn("No .env file found, using system environment variables")
		}
	})

	return &authService{
		userRepo:    repository.NewUserRepository(db),
		sessionRepo: repository.NewSessionRepository(db),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *authService) LoginWithProvider(provider domain.AuthProvider) (string, error) {
	config, err := a.getProviderConfig(provider)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %w", err)
	}

	// Generate a secure random state parameter
	state, err := a.generateSecureToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate state parameter: %w", err)
	}

	url := config.AuthCodeURL(state)
	return url, nil
}

func (a *authService) HandleProviderCallback(ctx context.Context, provider domain.AuthProvider, code string) (*domain.Session, error) {
	config, err := a.getProviderConfig(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider config: %w", err)
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		slog.Error("Failed to exchange code for token", "error", err)
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	userInfo, err := a.fetchGoogleUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	user, err := a.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	session, err := a.createSession(ctx, user.ID, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func (a *authService) getProviderConfig(provider domain.AuthProvider) (*oauth2.Config, error) {
	switch provider {
	case domain.AuthProviderGoogle:
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("missing Google OAuth credentials in environment")
		}

		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  "http://localhost:8080/auth/google/callback",
			Endpoint:     google.Endpoint,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// generateSecureToken generates a cryptographically secure random token
func (a *authService) generateSecureToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// googleUserInfo represents the user information from Google OAuth
type googleUserInfo struct {
	Sub     string `json:"sub"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Email   string `json:"email"`
}

// fetchGoogleUserInfo fetches user information from Google using the access token
func (a *authService) fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API returned status %d", res.StatusCode)
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// findOrCreateUser finds an existing user or creates a new one
func (a *authService) findOrCreateUser(ctx context.Context, userInfo *googleUserInfo) (*domain.User, error) {
	// Try to find existing user
	user, err := a.userRepo.FindByEmail(ctx, userInfo.Email)
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// User doesn't exist, create new one
	return a.createUser(ctx, userInfo)
}

// createUser creates a new user in the database
func (a *authService) createUser(ctx context.Context, userInfo *googleUserInfo) (*domain.User, error) {
	user := &domain.User{
		ID:         uuid.New().String(),
		Name:       userInfo.Name,
		Email:      userInfo.Email,
		Username:   userInfo.Sub, // Use Google's sub as username for now
		ProfilePic: userInfo.Picture,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := a.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (a *authService) createSession(ctx context.Context, userID string, provider domain.AuthProvider) (*domain.Session, error) {
	token, err := a.generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	session := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		Provider:  provider,
		ExpiredAt: time.Now().Add(sessionDuration),
		CreatedAt: time.Now(),
	}

	if err := a.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func (a *authService) ValidateSession(token string) (*domain.ValidateSessionDTO, *domain.AppError) {
	dto, err := a.sessionRepo.FindByTokenWithUser(context.Background(), token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.AppError{
				Code:    domain.ErrCodeAuthInvalidSession,
				Message: "Invalid or expired session",
				Err:     err,
			}
		}
		slog.Error("Failed to validate session", "error", err)
		return nil, &domain.AppError{
			Code:    domain.ErrCodeAuthInvalidSession,
			Message: "Session validation failed",
			Err:     err,
		}
	}

	return dto, nil
}

func (a *authService) Logout(ctx context.Context, token string) *domain.AppError {
	err := a.sessionRepo.DeleteByToken(ctx, token)
	if err != nil {
		slog.Error("Failed to logout", "error", err)
		return &domain.AppError{
			Code:    domain.ErrCodeAuthInvalidSession,
			Message: "Logout failed",
			Err:     err,
		}
	}
	return nil
}
