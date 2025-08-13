package main

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thantko20/tubbym-backend/internal/domain"
	"github.com/thantko20/tubbym-backend/internal/services"
)

func main() {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		return
	}
	defer db.Close()

	videoService := services.NewVideoService(db)

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/videos", func(c *fiber.Ctx) error {
		videos, count, err := videoService.GetVideos(c.Context(), nil)

		var domainErr *domain.Error

		if errors.As(err, &domainErr) {
			switch domainErr.Code {
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "Internal Server Error",
					"data":    domain.ErrInternalServer,
				})
			}
		}

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Internal Server Error",
				"code":    domain.ErrInternalServer,
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
			var domainErr *domain.Error
			if errors.As(err, &domainErr) {
				switch domainErr.Code {
				case services.ErrVideoNotFound:
					return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
						"success": false,
						"message": domainErr.Message,
						"code":    domainErr.Code,
					})
				default:
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"success": false,
						"message": "Internal Server Error",
						"data":    domain.ErrInternalServer,
					})
				}
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Internal Server Error",
				"data":    domain.ErrInternalServer,
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Video retrieved successfully",
			"data":    video,
			"count":   nil,
		})
	})

	app.Listen(":8080")
}
