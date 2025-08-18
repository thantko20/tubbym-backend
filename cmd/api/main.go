package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thantko20/tubbym-backend/internal/domain"
	"github.com/thantko20/tubbym-backend/internal/services"
	"github.com/thantko20/tubbym-backend/internal/storage"
)

func main() {
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		return
	}
	defer db.Close()

	store, err := storage.NewS3Storage("tubbym-test")
	if err != nil {
		slog.Error("Failed to create storage", "error", err)
		return
	}
	videoService := services.NewVideoService(db, store)

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/videos", func(c *fiber.Ctx) error {
		videos, count, err := videoService.GetVideos(c.Context(), nil)

		var domainErr *domain.AppError

		if errors.As(err, &domainErr) {
			switch domainErr.Code {
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Internal Server Error",
					"code":    domainErr.Code,
				})
			}
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
	})

	app.Get("/videos/:id", func(c *fiber.Ctx) error {
		video, err := videoService.GetVideoByID(c.Context(), c.Params("id"))
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
	})

	app.Post("/videos", func(c *fiber.Ctx) error {
		reqPayload := new(domain.CreateVideoReq)

		if err := c.BodyParser(reqPayload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request payload",
				"code":    domain.ErrCodeValidation,
			})
		}

		video, presignedUrl, err := videoService.CreateVideo(c.Context(), *reqPayload)

		if err != nil {
			fmt.Println("Error creating video:", err)
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
	})

	app.Post("/videos/:id/process", func(c *fiber.Ctx) error {
		videoId := c.Params("id")
		err := videoService.ProcessVideo(c.Context(), videoId)
		if err != nil {
			slog.Error("Failed to process video", "error", err)
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
	})

	app.Listen(":8080")
}
