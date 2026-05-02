package contact

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/serve"
)

// Service handles contact form submission business logic.
type Service interface {
	Submit(ctx context.Context, input ContactInput, ipAddress string) error
}

// ServiceConfig controls rate limiting and queue routing.
type ServiceConfig struct {
	RateLimitProfile ratelimit.RateLimitProfile
	QueueName        string
}

type contactService struct {
	repo    Repository
	limiter *ratelimit.Limiter
	queue   *queue.Dispatcher
	cfg     ServiceConfig
	logger  *slog.Logger
}

// NewService creates a Service with the given dependencies.
func NewService(repo Repository, limiter *ratelimit.Limiter, q *queue.Dispatcher, cfg ServiceConfig, logger *slog.Logger) *contactService {
	return &contactService{repo: repo, limiter: limiter, queue: q, cfg: cfg, logger: logger}
}

func (s *contactService) Submit(ctx context.Context, input ContactInput, ipAddress string) error {
	email := normalizeEmail(input.Email)
	clientIP := canonicalizeIP(ipAddress)

	if s.limiter == nil {
		return ErrRateLimitUnavailable
	}
	decision, err := s.limiter.Allow(ctx, s.cfg.RateLimitProfile, ratelimit.BuildKey(email, clientIP))
	if err != nil {
		return errors.Join(ErrRateLimitUnavailable, err)
	}
	if !decision.Allowed {
		return &RateLimitedError{RetryAfter: decision.RetryAfter}
	}

	sub := &Submission{
		Name:      input.Name,
		Email:     input.Email,
		Message:   input.Message,
		IPAddress: clientIP,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return fmt.Errorf("contact: persist submission: %w", err)
	}

	payload := NotificationPayload{
		SubmissionID: sub.ID,
		Name:         input.Name,
		Email:        input.Email,
		Message:      input.Message,
	}
	if err := s.queue.DispatchPayload(ctx, s.cfg.QueueName, payload); err != nil {
		s.logger.WarnContext(ctx, "contact: dispatch notification failed", "submission_id", sub.ID, "err", err)
	}

	s.logger.InfoContext(ctx, "contact: submission received", "submission_id", sub.ID)
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func canonicalizeIP(raw string) string {
	if normalized, ok := serve.NormalizeProxyIP(raw); ok {
		return normalized
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "unknown"
	}
	return raw
}
