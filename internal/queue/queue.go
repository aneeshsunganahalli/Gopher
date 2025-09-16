package queue

import (
	"context"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
)

type Queue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, job *types.Job) error

	// Dequeue removes and returns a job from the queue
	// This is a blocking operation that waits for jobs
	Dequeue(ctx context.Context) (*types.Job, error)

	// Size returns the current number of jobs in the queue
	Size(ctx context.Context) (int, error)

	// Health checks if the queue is healthy/reachable
	Health(ctx context.Context) error

	// Close closes the queue connection
	Close() error
}

type QueueStats struct {
	QueueSize int `json:"queue_size"`
	TotalEnqueued int `json:"total_enqueued"`
	TotalDequeued int `json:"total_dequeued"`
}
