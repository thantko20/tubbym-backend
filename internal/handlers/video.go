package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/thantko20/tubbym-backend/internal/auth"
	"github.com/thantko20/tubbym-backend/internal/domain"
	"github.com/thantko20/tubbym-backend/internal/services"
)

type Handlers struct {
	videoService services.VideoService
	authService  auth.AuthService
}

func NewHandlers(videoService services.VideoService, authService auth.AuthService) *Handlers {
	return &Handlers{
		videoService: videoService,
		authService:  authService,
	}
}

func (h *Handlers) GetVideos(c *fiber.Ctx) error {
	videos, count, err := h.videoService.GetVideos(c.Context(), nil)

	var domainErr *domain.AppError
	if errors.As(err, &domainErr) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal Server Error",
			"code":    domainErr.Code,
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal Server Error",
			"code":    9999,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Videos retrieved successfully",
		"data":    videos,
		"count":   count,
	})
}

func (h *Handlers) GetVideoByID(c *fiber.Ctx) error {
	video, err := h.videoService.GetVideoByID(c.Context(), c.Params("id"))
	if err != nil {
		var domainErr *domain.AppError
		if errors.As(err, &domainErr) {
			switch domainErr.Code {
			case domain.ErrCodeVideoNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"success": false,
					"message": domainErr.Message,
					"code":    domainErr.Code,
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Internal Server Error",
					"code":    domainErr.Code,
				})
			}
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal Server Error",
			"data":    9999,
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Video retrieved successfully",
		"data":    video,
		"count":   nil,
	})
}

func (h *Handlers) CreateVideo(c *fiber.Ctx) error {
	reqPayload := new(domain.CreateVideoReq)

	if err := c.BodyParser(reqPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request payload",
			"code":    domain.ErrCodeValidation,
		})
	}

	video, presignedUrl, err := h.videoService.CreateVideo(c.Context(), *reqPayload)
	if err != nil {
		var domainErr *domain.AppError
		if errors.As(err, &domainErr) {
			switch domainErr.Code {
			case domain.ErrCodeInvalidVideoData:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"success": false,
					"message": domainErr.Message,
					"code":    domainErr.Code,
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Internal Server Error",
					"code":    domainErr.Code,
				})
			}
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal Server Error",
			"code":    9999,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Video created successfully",
		"data": fiber.Map{
			"videoId":      video.ID,
			"presignedUrl": presignedUrl,
		},
	})
}

func (h *Handlers) ProcessVideo(c *fiber.Ctx) error {
	videoId := c.Params("id")
	err := h.videoService.ProcessVideo(c.Context(), videoId)
	if err != nil {
		var domainErr *domain.AppError
		if errors.As(err, &domainErr) {
			switch domainErr.Code {
			case domain.ErrCodeVideoNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"success": false,
					"message": domainErr.Message,
					"code":    domainErr.Code,
				})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Internal Server Error",
					"code":    domainErr.Code,
				})
			}
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal Server Error",
			"data":    9999,
		})
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"message": "Video processing started",
	})
}
