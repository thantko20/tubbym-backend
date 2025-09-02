package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thantko20/tubbym-backend/internal/domain"
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

	// SSE endpoint for video processing status
	app.Get("/videos/:id/status", handleVideoProcessingSSE(broker))

	app.Listen(":8080")
}

func handleVideoProcessingSSE(broker *pubsub.Broker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Set SSE headers
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Headers", "Cache-Control")

		videoID := c.Params("id")
		if videoID == "" {
			return c.Status(fiber.StatusBadRequest).SendString("event: error\ndata: {\"error\": \"Video ID is required\"}\n\n")
		}

		// Subscribe to the video processing topic
		topic := domain.GetVideoProcessingTopic(videoID)
		client := broker.Subscribe(topic)
		if client == nil {
			slog.Error("Failed to subscribe to topic", "topic", topic)
			return c.Status(fiber.StatusInternalServerError).SendString("event: error\ndata: {\"error\": \"Failed to subscribe to video processing updates\"}\n\n")
		}

		slog.Info("SSE client connected", "videoId", videoID)

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			defer broker.Unsubscribe(topic, client)
			defer func() {
				if r := recover(); r != nil {
					slog.Error("SSE connection panic recovered", "error", r, "stack", debug.Stack())
				}
			}()

			slog.Info("Starting SSE stream", "videoId", videoID)
			initialEvent := fmt.Sprintf("event: connected\ndata: {\"message\": \"Connected to video processing updates\", \"videoId\": \"%s\"}\n\n", videoID)
			w.WriteString(initialEvent)
			w.Flush()

			if client.Channel() == nil {
				slog.Error("Client channel is nil", "videoId", videoID)
				return
			}

			if client.Done() == nil {
				slog.Error("Client done channel is nil", "videoId", videoID)
				return
			}

			slog.Info("Listening channels")
			// Keep connection alive and send events
			for {
				select {

				case message := <-client.Channel():
					eventData := fmt.Sprintf("event: video_update\ndata: %s\n\n", message)
					if _, err := w.WriteString(eventData); err != nil {
						slog.Error("Failed to write SSE event", "error", err)
						return
					}
					if err := w.Flush(); err != nil {
						slog.Error("Failed to flush SSE event", "error", err)
						return
					}

				case <-client.Done():
					slog.Info("SSE client disconnected", "videoId", videoID)
					return

				case <-time.After(30 * time.Second):
					// Send keepalive ping every 30 seconds
					if _, err := w.WriteString("event: ping\ndata: {\"type\": \"keepalive\"}\n\n"); err != nil {
						slog.Error("Failed to write keepalive ping", "error", err)
						return
					}
					if err := w.Flush(); err != nil {
						slog.Error("Failed to flush keepalive ping", "error", err)
						return
					}
				}

			}
		})

		return nil
	}
}
