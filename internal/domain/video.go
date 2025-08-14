package domain

const (
	ErrCodeVideoNotFound      ErrorCode = 2001
	ErrCodeVideoInvalidID     ErrorCode = 2002
	ErrCodeVideoDatabaseError ErrorCode = 2003
)

type Video struct {
	ID           string `json:"id" db:"id"`
	Title        string `json:"title" db:"title"`
	Description  string `json:"description" db:"description"`
	Duration     int    `json:"duration" db:"duration"`
	Views        int    `json:"views"`
	Key          string `json:"key" db:"key"`
	ThumbnailKey string `json:"thumbnail_key" db:"thumbnail_key"`
	CreatedAt    string `json:"created_at" db:"created_at"`
	UpdatedAt    string `json:"updated_at" db:"updated_at"`
	DeletedAt    string `json:"deleted_at" db:"deleted_at"`
}

type VideoFilters struct {
	ID string `json:"id"`
}

type VideoService interface {
	GetVideoByID(id string) (*Video, error)
	GetVideos(filters VideoFilters) ([]Video, int, error)
}

type VideoReq struct {
	Title       string `json:"title" form:"title"`
	Description string `json:"description" form:"description"`
	Visibility  string `json:"visibility" form:"visibility"`
}
