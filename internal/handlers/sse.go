package handlers

import (
	"bufio"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thantko20/tubbym-backend/internal/domain"
	"github.com/thantko20/tubbym-backend/internal/pubsub"
)

func HandleVideoProcessingSSE(broker *pubsub.Broker) fiber.Handler {
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
