package services

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thantko20/tubbym-backend/internal/domain"
)

type videoService struct {
	db *sql.DB
}

func NewVideoService(db *sql.DB) *videoService {
	return &videoService{db: db}
}

func (s *videoService) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
	// Implementation to fetch video by ID

	videos, _, err := s.findVideos(ctx, &domain.VideoFilters{ID: id})

	if err != nil {
		return nil, err
	}

	if len(videos) == 0 {
		return nil, domain.NewAppError(domain.ErrCodeVideoNotFound, "Video not found", nil)
	}

	return &videos[0], nil
}

func (s *videoService) GetVideos(ctx context.Context, filters *domain.VideoFilters) ([]domain.Video, int, error) {
	videos, n, err := s.findVideos(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	return videos, n, nil
}

func (s *videoService) findVideos(ctx context.Context, filters *domain.VideoFilters) ([]domain.Video, int, error) {
	var videos []domain.Video
	var count int

	where := []string{"1 = 1"}
	var params []any

	if filters != nil {
		if filters.ID != "" {
			where = append(where, "id = ?")
			params = append(params, filters.ID)
		}
	}

	whereClause := strings.Join(where, " AND ")

	rows, err := s.db.QueryContext(ctx,
		`SELECT 
		id, title, description, duration, views, key, 
		thumbnail_key, created_at, updated_at, deleted_at 
		FROM videos WHERE `+whereClause, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var createdAt int64
	var updatedAt int64
	var deletedAt sql.NullInt64
	for rows.Next() {
		var video domain.Video
		if err := rows.Scan(&video.ID, &video.Title, &video.Description, &video.Duration, &video.Views, &video.Key, &video.ThumbnailKey,
			&createdAt, &updatedAt, &deletedAt); err != nil {
			return nil, 0, err
		}
		video.CreatedAt = time.Unix(createdAt, 0)
		video.UpdatedAt = time.Unix(updatedAt, 0)
		if deletedAt.Valid {
			video.DeletedAt = new(time.Time)
			*video.DeletedAt = time.Unix(deletedAt.Int64, 0)
		}
		videos = append(videos, video)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return videos, count, nil

}

func (s *videoService) CreateVideo(ctx context.Context, payload domain.CreateVideoReq) (*domain.Video, error) {

	validatedPayload, err := s.validateCreateVideoReq(&payload)
	if err != nil {
		return nil, err
	}

	newVideo := domain.Video{
		ID:          uuid.New().String(),
		Title:       validatedPayload.Title,
		Description: validatedPayload.Description,
		Visibility:  validatedPayload.Visibility,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err = s.insertVideo(ctx, newVideo); err != nil {
		return nil, err
	}

	return &newVideo, nil
}

func (s *videoService) insertVideo(ctx context.Context, video domain.Video) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO videos (id, title, description, duration, views, key, thumbnail_key, visibility, created_at, updated_at, deleted_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		video.ID, video.Title, video.Description, video.Duration, video.Views, video.Key, video.ThumbnailKey, video.Visibility, video.CreatedAt.Unix(), video.UpdatedAt.Unix(), nil)
	return err
}

func (s *videoService) validateCreateVideoReq(payload *domain.CreateVideoReq) (*domain.CreateVideoReq, error) {

	if payload.Title == "" {
		return nil, domain.NewAppError(domain.ErrCodeInvalidVideoData, "Video title is required", nil)
	}

	if payload.Description == "" {
		return nil, domain.NewAppError(domain.ErrCodeInvalidVideoData, "Video description is required", nil)
	}

	if payload.Visibility == "" {
		// defaults to public
		payload.Visibility = "public"
	}

	validVisibility := slices.Contains(
		[]domain.VideoVisibility{domain.VideoVisibilityPrivate, domain.VideoVisibilityPublic},
		payload.Visibility,
	)

	if !validVisibility {
		return nil, domain.NewAppError(domain.ErrCodeInvalidVideoData, "Video visibility must be either 'public' or 'private'", nil)
	}
	return payload, nil
}
