package domain

import (
	"encoding/json"
	"path/filepath"
	"time"
)

const (
	ErrCodeVideoNotFound      ErrorCode = 2001
	ErrCodeVideoInvalidID     ErrorCode = 2002
	ErrCodeVideoDatabaseError ErrorCode = 2003
	ErrCodeInvalidVideoData   ErrorCode = 2004
)

// CloudFront distribution URL for video streaming
// TODO: Move this to environment configuration
const CloudFrontDistributionURL = "https://d29kwr3nijxedo.cloudfront.net"

type VideoVisibility string

const (
	VideoVisibilityPublic  VideoVisibility = "public"
	VideoVisibilityPrivate VideoVisibility = "private"
)

type VideoStatus string

const (
	VideoStatusPendingUpload VideoStatus = "pending_upload"
	VideoStatusProcessing    VideoStatus = "processing"
	VideoStatusReady         VideoStatus = "ready"
	VideoStatusError         VideoStatus = "error"
)

// Video processing event types
type VideoProcessingEventType string

const (
	EventTypeVideoStatusUpdate        VideoProcessingEventType = "video:status:update"
	EventTypeVideoProcessingStarted   VideoProcessingEventType = "video:processing:started"
	EventTypeVideoProcessingCompleted VideoProcessingEventType = "video:processing:completed"
	EventTypeVideoProcessingError     VideoProcessingEventType = "video:processing:error"
)

// VideoProcessingEvent represents a video processing status update
type VideoProcessingEvent struct {
	VideoID   string                   `json:"videoId"`
	EventType VideoProcessingEventType `json:"eventType"`
	Status    VideoStatus              `json:"status"`
	Message   string                   `json:"message"`
	Progress  *int                     `json:"progress,omitempty"` // percentage (0-100)
	Error     string                   `json:"error,omitempty"`
	Timestamp time.Time                `json:"timestamp"`
}

// ToJSON converts the event to JSON string
func (e *VideoProcessingEvent) ToJSON() string {
	data, _ := json.Marshal(e)
	return string(data)
}

// GetVideoProcessingTopic returns the pubsub topic for a specific video
func GetVideoProcessingTopic(videoID string) string {
	return "video_processing:" + videoID
}

type Video struct {
	ID           string          `json:"id" db:"id"`
	Title        string          `json:"title" db:"title"`
	Description  string          `json:"description" db:"description"`
	Duration     int             `json:"duration" db:"duration"`
	Views        int             `json:"views" db:"views"`
	Key          string          `json:"key" db:"key"`
	ThumbnailKey string          `json:"thumbnailKey" db:"thumbnail_key"`
	Visibility   VideoVisibility `json:"visibility" db:"visibility"`
	Status       VideoStatus     `json:"status" db:"status"`
	URL          string          `json:"url" db:"-"` // CloudFront URL, not stored in DB
	// unix timestamp in db (integers)
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt" db:"deleted_at"`
}

// SetStreamingURL sets the CloudFront URL for the video based on its ID
func (v *Video) SetStreamingURL() {
	if v.Status == VideoStatusReady {
		v.URL = filepath.Join(CloudFrontDistributionURL, v.ID, "playlist.m3u8")
	}
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

func (r *CreateVideoReq) Validate() error {
	if r.Title == "" {
		return NewAppError(ErrCodeInvalidVideoData, "Video title is required", nil)
	}
	if r.Description == "" {
		return NewAppError(ErrCodeInvalidVideoData, "Video description is required", nil)
	}
	if r.Visibility == "" {
		r.Visibility = VideoVisibilityPublic // default to public
	}
	return nil
}

type ProcessVideoReq struct {
	VideoID string `json:"videoId"`
}

func (r *ProcessVideoReq) Validate() error {
	if r.VideoID == "" {
		return NewAppError(ErrCodeInvalidVideoData, "Video ID is required", nil)
	}
	return nil
}
