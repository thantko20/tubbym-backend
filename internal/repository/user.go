package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/thantko20/tubbym-backend/internal/domain"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, username, profile_pic, created_at, updated_at, deleted_at 
		FROM users 
		WHERE email = ? AND deleted_at IS NULL`

	row := r.db.QueryRowContext(ctx, query, email)

	var user domain.User
	var createdAt, updatedAt int64
	var deletedAt sql.NullInt64

	err := row.Scan(
		&user.ID, &user.Name, &user.Email, &user.Username, &user.ProfilePic,
		&createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	if deletedAt.Valid {
		deletedTime := time.Unix(deletedAt.Int64, 0)
		user.DeletedAt = &deletedTime
	}

	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, name, email, username, profile_pic, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Name, user.Email, user.Username, user.ProfilePic,
		user.CreatedAt.Unix(), user.UpdatedAt.Unix(),
	)
	return err
}
