# Video Processing Status Streaming (SSE)

This feature provides real-time updates about video processing status using Server-Sent Events (SSE).

## Overview

The video processing pipeline now publishes real-time status updates that clients can subscribe to via SSE. This allows frontend applications to display live progress information to users without polling the server.

## Architecture

### Components

1. **PubSub System** (`internal/pubsub/pubsub.go`)

   - In-memory message broker
   - Manages client subscriptions and message publishing
   - Thread-safe implementation with proper cleanup

2. **Video Processing Events** (`internal/domain/video.go`)

   - Structured event types for different processing stages
   - JSON-serializable event data
   - Progress tracking with percentage completion

3. **SSE Endpoint** (`/videos/{id}/status`)
   - Server-Sent Events endpoint for real-time updates
   - Per-video subscription using video ID
   - Automatic cleanup on client disconnect

### Event Types

- `video_status_update`: General status changes (processing → ready)
- `video_downloading`: Downloading original video from storage
- `video_transcoding`: Converting video to HLS format
- `video_uploading`: Uploading processed segments
- `video_error`: Error during any processing stage

## API Usage

### Start Video Processing

```bash
POST /videos/{id}/process
```

### Subscribe to Status Updates

```javascript
const eventSource = new EventSource("http://localhost:8080/videos/{id}/status");

eventSource.addEventListener("video_update", function (event) {
  const data = JSON.parse(event.data);
  console.log("Processing update:", data);
});
```

### Event Data Structure

```json
{
  "videoId": "12345",
  "eventType": "video_transcoding",
  "status": "processing",
  "message": "Transcoding video to HLS format",
  "progress": 75,
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Implementation Details

### Video Service Changes

The video service now:

- Publishes events at each processing stage
- Includes progress updates for upload operations
- Handles error cases with detailed error messages
- Updates database status atomically with event publishing

### Error Handling

- Network disconnections are handled gracefully
- Failed processing stages publish error events
- Database is updated to reflect error states
- Clients receive detailed error information

### Performance Considerations

- Buffered channels prevent blocking publishers
- Automatic client cleanup prevents memory leaks
- Keepalive pings maintain connection health
- Maximum event history limits memory usage

## Testing

Use the included test client (`test-sse.html`) to:

1. Connect to a video's processing stream
2. Trigger video processing via API
3. Watch real-time updates in the browser

### Example Workflow

1. Create a video: `POST /videos`
2. Upload video file using presigned URL
3. Start processing: `POST /videos/{id}/process`
4. Connect SSE client: `GET /videos/{id}/status`
5. Monitor real-time progress updates

## Configuration

The SSE implementation uses these defaults:

- 30-second keepalive interval
- 10-message channel buffer
- Automatic client timeout handling

## Future Enhancements

- Redis-based pubsub for horizontal scaling
- Event persistence for replay functionality
- WebSocket alternative for bidirectional communication
- Metrics and monitoring integration
