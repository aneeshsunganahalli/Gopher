package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"go.uber.org/zap"
)

// ImageJobHandler handles image processing jobs
type ImageJobHandler struct {
	logger *zap.Logger
}

// ImagePayload represents the payload for image processing jobs
type ImagePayload struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

func NewImageJobHandler(logger *zap.Logger) *ImageJobHandler {
	return &ImageJobHandler{logger: logger}
}

func (h *ImageJobHandler) Type() string {
	return "image_resize"
}

func (h *ImageJobHandler) Description() string {
	return "Resizes images to specified dimensions"
}

func (h *ImageJobHandler) Handle(ctx context.Context, job *types.Job) error {
	// Parse payload
	var payload ImagePayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("invalid image payload: %w", err)
	}
	
	// Validate required fields
	if payload.URL == "" {
		return fmt.Errorf("image URL cannot be empty")
	}
	if payload.Width <= 0 || payload.Height <= 0 {
		return fmt.Errorf("image dimensions must be positive")
	}
	
	h.logger.Info("Processing image",
		zap.String("job_id", job.ID),
		zap.String("url", payload.URL),
		zap.Int("width", payload.Width),
		zap.Int("height", payload.Height),
	)
	
	// Simulate CPU-intensive image processing
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		// Image processed successfully
	}
	
	h.logger.Info("Image processed successfully",
		zap.String("job_id", job.ID),
		zap.String("url", payload.URL),
	)
	
	return nil
}