package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"go.uber.org/zap"
)

type EmailJobHandler struct {
	logger *zap.Logger
}


// EmailPayload represents the payload for email jobs
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func NewEmailJobHandler(logger *zap.Logger) *EmailJobHandler {
	return &EmailJobHandler{logger: logger}
}

func (h *EmailJobHandler) Type() string {
	return "email"
}

func (h *EmailJobHandler) Description() string {
	return "Sends emails to specified recipients"
}

func (h *EmailJobHandler) Handle(ctx context.Context, job *types.Job) error {
	// Parse payload
	var payload EmailPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("invalid email payload: %w", err)
	}
	
	// Validate required fields
	if payload.To == "" {
		return fmt.Errorf("email recipient cannot be empty")
	}
	if payload.Subject == "" {
		return fmt.Errorf("email subject cannot be empty")
	}
	
	h.logger.Info("Sending email",
		zap.String("job_id", job.ID),
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
	)
	
	// Simulate email sending with some processing time
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		// Email "sent" successfully
	}
	
	h.logger.Info("Email sent successfully",
		zap.String("job_id", job.ID),
		zap.String("to", payload.To),
	)
	
	return nil
}
