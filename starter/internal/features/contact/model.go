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

// RateLimitedError preserves retry metadata for contact-form limit denials.
type RateLimitedError struct {
	RetryAfter time.Duration
}

// ErrRateLimited is returned when a submitter has exceeded the allowed rate.
var ErrRateLimited = errors.New("contact: rate limit exceeded")

// ErrRateLimitUnavailable is returned when the backing store for rate limiting
// cannot be consulted safely, so submissions are rejected temporarily.
var ErrRateLimitUnavailable = errors.New("contact: rate limit unavailable")

func (e *RateLimitedError) Error() string {
	return ErrRateLimited.Error()
}

func (e *RateLimitedError) Unwrap() error {
	return ErrRateLimited
}

// RateLimitRetryAfter extracts retry metadata from err when available.
func RateLimitRetryAfter(err error) time.Duration {
	var rateErr *RateLimitedError
	if errors.As(err, &rateErr) {
		return rateErr.RetryAfter
	}
	return 0
}
