package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/thantko20/tubbym-backend/internal/domain"
)

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	FindByTokenWithUser(ctx context.Context, token string) (*domain.ValidateSessionDTO, error)
	DeleteByToken(ctx context.Context, token string) error
}

type sessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, provider, expired_at, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.Token, session.Provider,
		session.ExpiredAt.Unix(), session.CreatedAt.Unix(),
	)
	return err
}

func (r *sessionRepository) FindByTokenWithUser(ctx context.Context, token string) (*domain.ValidateSessionDTO, error) {
	query := `
		SELECT 
			s.id, s.user_id, s.token, s.provider, s.expired_at, s.created_at, s.deleted_at,
			u.id, u.name, u.email, u.username, u.profile_pic, u.created_at, u.updated_at, u.deleted_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token = ? AND s.deleted_at IS NULL AND s.expired_at > ?`

	row := r.db.QueryRowContext(ctx, query, token, time.Now().Unix())

	var dto domain.ValidateSessionDTO
	var sessionCreatedAt, sessionExpiredAt int64
	var sessionDeletedAt sql.NullInt64
	var userCreatedAt, userUpdatedAt int64
	var userDeletedAt sql.NullInt64

	err := row.Scan(
		&dto.Session.ID, &dto.Session.UserID, &dto.Session.Token, &dto.Session.Provider,
		&sessionExpiredAt, &sessionCreatedAt, &sessionDeletedAt,
		&dto.User.ID, &dto.User.Name, &dto.User.Email, &dto.User.Username, &dto.User.ProfilePic,
		&userCreatedAt, &userUpdatedAt, &userDeletedAt,
	)

	if err != nil {
		return nil, err
	}

	// Convert timestamps
	dto.Session.ExpiredAt = time.Unix(sessionExpiredAt, 0)
	dto.Session.CreatedAt = time.Unix(sessionCreatedAt, 0)
	if sessionDeletedAt.Valid {
		deletedTime := time.Unix(sessionDeletedAt.Int64, 0)
		dto.Session.DeletedAt = &deletedTime
	}

	dto.User.CreatedAt = time.Unix(userCreatedAt, 0)
	dto.User.UpdatedAt = time.Unix(userUpdatedAt, 0)
	if userDeletedAt.Valid {
		deletedTime := time.Unix(userDeletedAt.Int64, 0)
		dto.User.DeletedAt = &deletedTime
	}

	return &dto, nil
}

func (r *sessionRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `
		UPDATE sessions 
		SET deleted_at = ? 
		WHERE token = ? AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, time.Now().Unix(), token)
	return err
}
