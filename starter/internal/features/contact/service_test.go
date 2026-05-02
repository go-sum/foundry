package contact

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
)

// fakeRepo is a manual implementation of Repository for service tests.
type fakeRepo struct {
	err     error
	created *Submission
	called  bool
}

func (f *fakeRepo) Create(_ context.Context, s *Submission) error {
	f.called = true
	f.created = s
	if f.err != nil {
		return f.err
	}
	s.ID = "test-id-123"
	s.CreatedAt = time.Now()
	return nil
}

type errorStore struct {
	err error
}

const testRateLimitProfile = "contact.submit.email"

func (s errorStore) Allow(_ context.Context, _ string, _ ratelimit.Policy) (ratelimit.Decision, error) {
	return ratelimit.Decision{}, s.err
}

func defaultConfig() ServiceConfig {
	return ServiceConfig{
		RateLimitProfile: testRateLimitProfile,
		QueueName:        QueueName,
	}
}

func newLimiter(t *testing.T, store ratelimit.Store, capacity int) *ratelimit.Limiter {
	t.Helper()
	limiter, err := ratelimit.New(ratelimit.Config{
		Store: store,
		Profiles: map[ratelimit.RateLimitProfile]ratelimit.Policy{
			testRateLimitProfile: {
				Capacity:  capacity,
				RefillPer: time.Minute,
			},
		},
	})
	if err != nil {
		t.Fatalf("ratelimit.New() error = %v", err)
	}
	return limiter
}

func newMemoryLimiter(t *testing.T, capacity int) *ratelimit.Limiter {
	t.Helper()
	return newLimiter(t, ratelimit.NewMemoryStore(ratelimit.MemoryStoreConfig{}), capacity)
}

func newTestService(repo Repository, limiter *ratelimit.Limiter, q *queue.Dispatcher) Service {
	return NewService(repo, limiter, q, defaultConfig(), slog.Default())
}

func exhaustLimiterKey(t *testing.T, limiter *ratelimit.Limiter, key string, capacity int) {
	t.Helper()
	for i := 0; i < capacity; i++ {
		decision, err := limiter.Allow(context.Background(), testRateLimitProfile, key)
		if err != nil {
			t.Fatalf("limiter.Allow(%d) error = %v", i, err)
		}
		if !decision.Allowed {
			t.Fatalf("limiter.Allow(%d) denied unexpectedly", i)
		}
	}
}

func TestService_Submit_Success(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newMemoryLimiter(t, 3)
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	svc := newTestService(repo, limiter, q)

	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	if err := svc.Submit(context.Background(), input, "127.0.0.1"); err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}

	if !repo.called {
		t.Error("expected repo.Create to be called")
	}
	if repo.created.Name != "Alice" {
		t.Errorf("repo.Create Name = %q, want %q", repo.created.Name, "Alice")
	}
	if repo.created.Email != "alice@example.com" {
		t.Errorf("repo.Create Email = %q, want %q", repo.created.Email, "alice@example.com")
	}
	if repo.created.Message != "Hello" {
		t.Errorf("repo.Create Message = %q, want %q", repo.created.Message, "Hello")
	}
	if repo.created.IPAddress != "127.0.0.1" {
		t.Errorf("repo.Create IPAddress = %q, want %q", repo.created.IPAddress, "127.0.0.1")
	}
}

func TestService_Submit_RateLimited(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newMemoryLimiter(t, 3)
	q := queue.NewDispatcher(nil)

	exhaustLimiterKey(t, limiter, ratelimit.BuildKey("alice@example.com", "127.0.0.1"), 3)

	svc := newTestService(repo, limiter, q)
	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}

	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
	var rlErr *RateLimitedError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitedError, got %T", err)
	}
	if rlErr.RetryAfter <= 0 {
		t.Fatalf("RetryAfter = %v, want > 0", rlErr.RetryAfter)
	}
	if repo.called {
		t.Error("repo.Create must NOT be called when rate limited")
	}
}

func TestService_Submit_RepoError(t *testing.T) {
	repoErr := errors.New("db: connection refused")
	repo := &fakeRepo{err: repoErr}
	limiter := newMemoryLimiter(t, 3)
	q := queue.NewDispatcher(nil)

	svc := newTestService(repo, limiter, q)

	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error from repo failure, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected error chain to contain repoErr, got: %v", err)
	}
}

func TestService_Submit_QueueError_StillSucceeds(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newMemoryLimiter(t, 3)
	q := queue.NewDispatcher(nil)

	svc := newTestService(repo, limiter, q)

	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if err != nil {
		t.Fatalf("Submit must return nil even when queue dispatch fails, got: %v", err)
	}
	if !repo.called {
		t.Error("expected repo.Create to be called despite queue error")
	}
}

func TestService_Submit_RateLimitUnavailable_FailsClosed(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newLimiter(t, errorStore{err: errors.New("redis unavailable")}, 3)
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	svc := newTestService(repo, limiter, q)

	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if !errors.Is(err, ErrRateLimitUnavailable) {
		t.Fatalf("expected ErrRateLimitUnavailable, got: %v", err)
	}
	if repo.called {
		t.Error("repo.Create must NOT be called when the rate limiter is unavailable")
	}
}

func TestService_Submit_EmailNormalized(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newMemoryLimiter(t, 3)
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	exhaustLimiterKey(t, limiter, ratelimit.BuildKey("alice@example.com", "127.0.0.1"), 3)

	svc := newTestService(repo, limiter, q)
	input := ContactInput{
		Name:    "Alice",
		Email:   "  ALICE@EXAMPLE.COM  ",
		Message: "Hello",
	}

	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited for normalized email, got: %v", err)
	}
}

func TestService_Submit_BelowRateLimit(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newMemoryLimiter(t, 3)
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	exhaustLimiterKey(t, limiter, ratelimit.BuildKey("alice@example.com", "127.0.0.1"), 2)

	svc := newTestService(repo, limiter, q)
	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	if err := svc.Submit(context.Background(), input, "127.0.0.1"); err != nil {
		t.Fatalf("expected success when below rate limit, got: %v", err)
	}
}

func TestService_Submit_RateLimitKeyIsEmailAndIP(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newMemoryLimiter(t, 1)
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	exhaustLimiterKey(t, limiter, ratelimit.BuildKey("alice@example.com", "127.0.0.1"), 1)

	svc := newTestService(repo, limiter, q)
	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}

	// Same email but different IP — different composite key, should be allowed.
	err := svc.Submit(context.Background(), input, "198.51.100.9")
	if err != nil {
		t.Fatalf("expected success for different IP, got: %v", err)
	}

	// Same email and same IP — same composite key, should be rate limited.
	err = svc.Submit(context.Background(), input, "127.0.0.1")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited for same IP, got: %v", err)
	}
}

func TestService_Submit_PersistsCanonicalIP(t *testing.T) {
	repo := &fakeRepo{}
	limiter := newMemoryLimiter(t, 3)
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	svc := newTestService(repo, limiter, q)
	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	if err := svc.Submit(context.Background(), input, "127.0.0.1:1234"); err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}
	if repo.created.IPAddress != "127.0.0.1" {
		t.Errorf("repo.Create IPAddress = %q, want %q", repo.created.IPAddress, "127.0.0.1")
	}
}
