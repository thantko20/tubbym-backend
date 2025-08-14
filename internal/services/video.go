package services

import (
	"context"
	"database/sql"
	"strings"

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

	for rows.Next() {
		var video domain.Video
		if err := rows.Scan(&video.ID, &video.Title, &video.Description, &video.Duration, &video.Views, &video.Key, &video.ThumbnailKey, &video.CreatedAt, &video.UpdatedAt, &video.DeletedAt); err != nil {
			return nil, 0, err
		}
		videos = append(videos, video)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return videos, count, nil

}
