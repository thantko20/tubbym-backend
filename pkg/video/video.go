package video

import (
	"database/sql"

	"github.com/thantko20/tubbym-backend/tubbym"
)

const (
	ErrVideoNotFound = 2001 + iota
	ErrVideoInvalidID
	ErrVideoDatabaseError
)

type service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *service {
	return &service{db: db}
}

func (s *service) GetVideoByID(id string) (*tubbym.Video, error) {
	// Implementation to fetch video by ID
	return nil, &tubbym.Error{
		Message: "Video not found",
		Code:    ErrVideoNotFound,
		Action:  "GetVideoByID",
	}
}

func (s *service) GetVideos(filters *tubbym.VideoFilters) ([]tubbym.Video, int, error) {
	return nil, 0, nil
}
