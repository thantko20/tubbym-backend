package main

import (
	"database/sql"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thantko20/tubbym-backend/internal/auth"
	"github.com/thantko20/tubbym-backend/internal/handlers"
	"github.com/thantko20/tubbym-backend/internal/pubsub"
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

	// Create pubsub broker
	broker := pubsub.NewBroker()
	defer broker.Close()

	videoService := services.NewVideoService(db, store, broker)
	authService := auth.NewAuthService(db)

	// Create handlers
	h := handlers.NewHandlers(videoService, authService)

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// Video routes
	app.Get("/videos", h.GetVideos)
	app.Get("/videos/:id", h.GetVideoByID)
	app.Post("/videos", h.CreateVideo)
	app.Post("/videos/:id/process", h.ProcessVideo)
	app.Get("/videos/:id/status", handlers.HandleVideoProcessingSSE(broker))

	// Auth routes
	app.Get("/auth/:provider/login", h.LoginWithProvider)
	app.Get("/auth/:provider/callback", h.HandleProviderCallback)
	app.Post("/auth/logout", h.Logout)

	app.Listen(":8080")
}
