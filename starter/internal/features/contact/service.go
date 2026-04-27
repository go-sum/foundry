package contact

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/queue"
)

// Service handles contact form submission business logic.
type Service interface {
	Submit(ctx context.Context, input ContactInput, ipAddress string) error
}

// ServiceConfig controls rate limiting and queue routing.
type ServiceConfig struct {
	RateLimit  int
	RateWindow time.Duration
	QueueName  string
}

type contactService struct {
	repo   Repository
	kv     kv.Store
	queue  *queue.Dispatcher
	cfg    ServiceConfig
	logger *slog.Logger
}

// NewService creates a Service with the given dependencies.
func NewService(repo Repository, store kv.Store, q *queue.Dispatcher, cfg ServiceConfig, logger *slog.Logger) *contactService {
	return &contactService{repo: repo, kv: store, queue: q, cfg: cfg, logger: logger}
}

func (s *contactService) Submit(ctx context.Context, input ContactInput, ipAddress string) error {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	key := "contact:rate:" + email

	count, err := s.readCount(ctx, key)
	if err != nil {
		s.logger.WarnContext(ctx, "contact: kv read failed, proceeding without rate limit", "err", err)
	} else if count >= s.cfg.RateLimit {
		return ErrRateLimited
	}

	sub := &Submission{
		Name:      input.Name,
		Email:     input.Email,
		Message:   input.Message,
		IPAddress: ipAddress,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return fmt.Errorf("contact: persist submission: %w", err)
	}

	if err := s.writeCount(ctx, key, count+1); err != nil {
		s.logger.WarnContext(ctx, "contact: kv write failed", "err", err)
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

func (s *contactService) readCount(ctx context.Context, key string) (int, error) {
	b, err := s.kv.Get(ctx, key)
	if err != nil {
		if errors.Is(err, kv.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if len(b) < 8 {
		return 0, nil
	}
	return int(binary.BigEndian.Uint64(b)), nil
}

func (s *contactService) writeCount(ctx context.Context, key string, count int) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(count))
	return s.kv.Set(ctx, key, b, kv.SetOptions{TTL: s.cfg.RateWindow})
}
