package main

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thantko20/tubbym-backend/pkg/video"
	"github.com/thantko20/tubbym-backend/tubbym"
)

func main() {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		return
	}
	defer db.Close()

	videoService := video.NewService(db)

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/videos", func(c *fiber.Ctx) error {
		videos, n, err := videoService.GetVideos(nil)

		if errors.As(err, tubbym.Error{}) {
			tubbymErr := err.(*tubbym.Error)
			slog.Error("Error fetching videos", "error", tubbymErr)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  tubbymErr.Message,
				"code":   tubbymErr.Code,
				"action": tubbymErr.Action,
			})
		}

		return c.JSON(fiber.Map{
			"videos": videos,
			"count":  n,
		})
	})

	app.Listen(":8080")
}
