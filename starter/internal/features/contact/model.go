package contact

import (
	"errors"
	"time"
)

// Submission is a persisted contact form entry.
type Submission struct {
	ID        string
	Name      string
	Email     string
	Message   string
	IPAddress string
	CreatedAt time.Time
}

// ContactInput holds validated form input.
type ContactInput struct {
	Name    string `form:"name"    json:"name"    validate:"required,max=100"`
	Email   string `form:"email"   json:"email"   validate:"required,email,max=255"`
	Message string `form:"message" json:"message" validate:"required,max=5000"`
}

// NotificationPayload is the JSON payload enqueued for async notification.
type NotificationPayload struct {
	SubmissionID string `json:"submission_id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Message      string `json:"message"`
}

// ErrRateLimited is returned when a submitter has exceeded the allowed rate.
var ErrRateLimited = errors.New("contact: rate limit exceeded")
