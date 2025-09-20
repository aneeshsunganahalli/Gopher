package types

import (
	"time"
)

// ScheduledJob represents a job that will be executed at a future time
type ScheduledJob struct {
	Job            *Job      `json:"job"`
	ExecuteAt      time.Time `json:"execute_at"`
	Recurring      bool      `json:"recurring"`
	CronExpression string    `json:"cron_expression,omitempty"`
}

// FailedJobInfo contains information about a failed job in the DLQ
type FailedJobInfo struct {
	Job      *Job      `json:"job"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
}
