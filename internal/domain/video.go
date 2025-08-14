package domain

import "time"

const (
	ErrCodeVideoNotFound      ErrorCode = 2001
	ErrCodeVideoInvalidID     ErrorCode = 2002
	ErrCodeVideoDatabaseError ErrorCode = 2003
	ErrCodeInvalidVideoData   ErrorCode = 2004
)

type VideoVisibility string

const (
	VideoVisibilityPublic  VideoVisibility = "public"
	VideoVisibilityPrivate VideoVisibility = "private"
)

type Video struct {
	ID           string          `json:"id" db:"id"`
	Title        string          `json:"title" db:"title"`
	Description  string          `json:"description" db:"description"`
	Duration     int             `json:"duration" db:"duration"`
	Views        int             `json:"views" db:"views"`
	Key          string          `json:"key" db:"key"`
	ThumbnailKey string          `json:"thumbnailKey" db:"thumbnail_key"`
	Visibility   VideoVisibility `json:"visibility" db:"visibility"`
	// unix timestamp in db (integers)
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt" db:"deleted_at"`
}

type VideoFilters struct {
	ID string `json:"id"`
}

type VideoService interface {
	GetVideoByID(id string) (*Video, error)
	GetVideos(filters VideoFilters) ([]Video, int, error)
}

type CreateVideoReq struct {
	Title       string          `json:"title" form:"title"`
	Description string          `json:"description" form:"description"`
	Visibility  VideoVisibility `json:"visibility" form:"visibility"`
}
