package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thantko20/tubbym-backend/internal/domain"
	"github.com/thantko20/tubbym-backend/internal/storage"
	"github.com/thantko20/tubbym-backend/internal/transcoder"
)

type videoService struct {
	db      *sql.DB
	storage storage.Storage
}

func NewVideoService(db *sql.DB, storage storage.Storage) *videoService {
	return &videoService{db: db, storage: storage}
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
		thumbnail_key, created_at, updated_at, deleted_at,
		status
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
			&createdAt, &updatedAt, &deletedAt, &video.Status); err != nil {
			return nil, 0, err
		}
		video.CreatedAt = time.Unix(createdAt, 0)
		video.UpdatedAt = time.Unix(updatedAt, 0)
		if deletedAt.Valid {
			video.DeletedAt = new(time.Time)
			*video.DeletedAt = time.Unix(deletedAt.Int64, 0)
		}
		// Set the streaming URL for ready videos
		video.SetStreamingURL()
		videos = append(videos, video)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return videos, count, nil

}

func (s *videoService) CreateVideo(ctx context.Context, payload domain.CreateVideoReq) (*domain.Video, string, error) {

	err := payload.Validate()
	if err != nil {
		return nil, "", err
	}

	var id = uuid.New().String()

	newVideo := domain.Video{
		ID:          id,
		Title:       payload.Title,
		Description: payload.Description,
		Visibility:  payload.Visibility,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Key:         filepath.Join("raw-videos", fmt.Sprintf("%s.mp4", id)),
	}

	if err = s.insertVideo(ctx, newVideo); err != nil {
		return nil, "", err
	}

	presignedURL, err := s.storage.GetPresignedURL(ctx, newVideo.Key)
	if err != nil {
		return nil, "", err
	}

	return &newVideo, presignedURL, nil
}

func (s *videoService) insertVideo(ctx context.Context, video domain.Video) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO videos (id, title, description, duration, views, key, thumbnail_key, visibility, created_at, updated_at, deleted_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		video.ID, video.Title, video.Description, video.Duration, video.Views, video.Key, video.ThumbnailKey, video.Visibility, video.CreatedAt.Unix(), video.UpdatedAt.Unix(), nil)
	return err
}

func (s *videoService) ProcessVideo(ctx context.Context, videoId string) error {
	video, err := s.GetVideoByID(ctx, videoId)
	if err != nil {
		return domain.NewAppError(domain.ErrCodeVideoNotFound, "Video not found", nil)
	}

	slog.Info("starting video processing", "videoId", video.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE videos SET status = ?, updated_at = ? WHERE id = ?`, domain.VideoStatusProcessing, time.Now().Unix(), video.ID)

	if err != nil {
		slog.Error("failed to update video status", "error", err)
		return fmt.Errorf("failed to update video status: %w", err)
	}

	go func() {
		videoName := fmt.Sprintf("%s.mp4", video.ID)
		tmpDir := filepath.Join(os.TempDir(), "tubbym-backend")
		rawDir := filepath.Join(tmpDir, "raw-videos")
		processedDir := filepath.Join(tmpDir, "processed-videos")
		dst := filepath.Join(rawDir, videoName)
		if err := os.MkdirAll(rawDir, 0755); err != nil {
			slog.Error("failed to create tmp raw-videos directory", "error", err)
			return
		}

		if err := os.MkdirAll(processedDir, 0755); err != nil {
			slog.Error("failed to create tmp processed-videos directory", "error", err)
			return
		}

		slog.Info("downloading video", "videoName", videoName)
		err = s.storage.Download(context.TODO(), "raw-videos/"+videoName, dst)
		if err != nil {
			slog.Error("failed to download video", "error", err)
			return
		}
		defer s.storage.Cleanup(context.TODO(), dst)

		t := transcoder.New("ffmpeg", tmpDir)
		slog.Info("starting video transcoding", "videoId", video.ID)

		transcodingStart := time.Now()

		outputDir, err := t.TranscodeToHLS(dst)
		transcodingElapsed := time.Since(transcodingStart)
		slog.Info("video transcoding completed", "videoId", video.ID, "duration", transcodingElapsed)
		if err != nil {
			slog.Error("failed to transcode video", "error", err)
			return
		}
		entries, err := os.ReadDir(outputDir)
		if err != nil {
			slog.Error("failed to read output directory", "error", err)
			return
		}

		slog.Info("uploading transcoded video segments", "videoId", video.ID)
		for _, entry := range entries {
			path := filepath.Join(outputDir, entry.Name())
			err := s.storage.Upload(context.TODO(), filepath.Join("processed-videos", video.ID, entry.Name()), path)
			if err != nil {
				slog.Error("failed to upload video", "error", err)
				return
			}
		}
		slog.Info("video transcoding completed", "videoId", video.ID)

		_, err = s.db.ExecContext(ctx, `UPDATE videos SET status = ?, updated_at = ? WHERE id = ?`, domain.VideoStatusReady, time.Now().Unix(), video.ID)
		if err != nil {
			slog.Error("failed to update video status to ready", "error", err)
			return
		}
	}()

	return err
}
