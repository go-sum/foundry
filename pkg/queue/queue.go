package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("queue: job not found")
	ErrQueueUnknown = errors.New("queue: unknown queue name")
	ErrClosed       = errors.New("queue: already closed")
)

type Priority int

const (
	PriorityCritical Priority = 0
	PriorityHigh     Priority = 10
	PriorityDefault  Priority = 20
	PriorityLow      Priority = 30
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusDead      Status = "dead"
)

type Job struct {
	ID          string
	Queue       string
	Priority    Priority
	Payload     json.RawMessage
	Status      Status
	Attempts    int
	MaxAttempts int
	LastError   string
	RunAt       time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// HandlerFunc processes a single job. Return non-nil to trigger retry.
type HandlerFunc func(ctx context.Context, job Job) error
