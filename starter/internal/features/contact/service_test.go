package contact

import (
	"context"
	"encoding/binary"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/go-sum/kv"
	"github.com/go-sum/queue"
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
	// Simulate DB assigning an ID.
	s.ID = "test-id-123"
	s.CreatedAt = time.Now()
	return nil
}

// fakeKV is a manual implementation of kv.Store for service tests.
type fakeKV struct {
	data    map[string][]byte
	getErr  error
	setErr  error
	getCalled bool
	setCalled bool
	lastSetKey string
	lastSetVal []byte
}

func newFakeKV() *fakeKV {
	return &fakeKV{data: make(map[string][]byte)}
}

func (f *fakeKV) Ping(_ context.Context) error { return nil }

func (f *fakeKV) Get(_ context.Context, key string) ([]byte, error) {
	f.getCalled = true
	if f.getErr != nil {
		return nil, f.getErr
	}
	v, ok := f.data[key]
	if !ok {
		return nil, kv.ErrNotFound
	}
	return v, nil
}

func (f *fakeKV) Set(_ context.Context, key string, value []byte, _ kv.SetOptions) error {
	f.setCalled = true
	f.lastSetKey = key
	f.lastSetVal = value
	if f.setErr != nil {
		return f.setErr
	}
	f.data[key] = value
	return nil
}

func (f *fakeKV) Delete(_ context.Context, _ ...string) error { return nil }
func (f *fakeKV) Exists(_ context.Context, _ ...string) (int64, error) { return 0, nil }
func (f *fakeKV) Close() error { return nil }

// setCount stores n as a big-endian uint64 for the given key, mimicking
// the encoding used by contactService.writeCount.
func (f *fakeKV) setCount(key string, n int) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(n))
	f.data[key] = b
}

func defaultConfig() ServiceConfig {
	return ServiceConfig{
		RateLimit:  3,
		RateWindow: time.Minute,
		QueueName:  QueueName,
	}
}

func newTestService(repo Repository, store kv.Store, q *queue.Dispatcher) Service {
	return NewService(repo, store, q, defaultConfig(), slog.Default())
}

// TestService_Submit_Success verifies the happy path: kv miss, repo creates, queue dispatches.
func TestService_Submit_Success(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeKV()
	q := queue.NewDispatcher(nil)
	// Register a no-op handler so sync-mode dispatch succeeds.
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	svc := newTestService(repo, store, q)

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
	if !store.setCalled {
		t.Error("expected kv.Set to be called to increment rate-limit counter")
	}
}

// TestService_Submit_RateLimited verifies that a kv count at the limit causes
// ErrRateLimited to be returned before the repo is called.
func TestService_Submit_RateLimited(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeKV()
	q := queue.NewDispatcher(nil)

	cfg := ServiceConfig{
		RateLimit:  3,
		RateWindow: time.Minute,
		QueueName:  QueueName,
	}
	svc := NewService(repo, store, q, cfg, slog.Default())

	// Seed the rate-limit counter at the limit.
	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	key := "contact:rate:alice@example.com"
	store.setCount(key, cfg.RateLimit)

	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
	if repo.called {
		t.Error("repo.Create must NOT be called when rate limited")
	}
}

// TestService_Submit_RepoError verifies that a repo failure is propagated.
func TestService_Submit_RepoError(t *testing.T) {
	repoErr := errors.New("db: connection refused")
	repo := &fakeRepo{err: repoErr}
	store := newFakeKV()
	q := queue.NewDispatcher(nil)

	svc := newTestService(repo, store, q)

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

// TestService_Submit_QueueError_StillSucceeds verifies that a queue dispatch
// failure does NOT cause Submit to return an error (fire-and-forget).
func TestService_Submit_QueueError_StillSucceeds(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeKV()
	// Dispatcher with nil store (sync mode) but no handler registered for the
	// queue name — DispatchPayload returns queue.ErrQueueUnknown.
	q := queue.NewDispatcher(nil)

	svc := newTestService(repo, store, q)

	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if err != nil {
		t.Fatalf("Submit must return nil even when queue dispatch fails, got: %v", err)
	}
	// Repo must still have been called.
	if !repo.called {
		t.Error("expected repo.Create to be called despite queue error")
	}
}

// TestService_Submit_KVError_Degraded verifies that a kv.Get error causes the
// service to proceed without rate limiting (submission still succeeds).
func TestService_Submit_KVError_Degraded(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeKV()
	store.getErr = errors.New("kv: connection refused")
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	svc := newTestService(repo, store, q)

	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	err := svc.Submit(context.Background(), input, "127.0.0.1")
	if err != nil {
		t.Fatalf("Submit must succeed in degraded mode, got: %v", err)
	}
	if !repo.called {
		t.Error("expected repo.Create to be called in degraded KV mode")
	}
}

// TestService_Submit_EmailNormalized verifies that the rate-limit key uses a
// lowercased, trimmed version of the email.
func TestService_Submit_EmailNormalized(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeKV()
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	cfg := ServiceConfig{
		RateLimit:  3,
		RateWindow: time.Minute,
		QueueName:  QueueName,
	}
	svc := NewService(repo, store, q, cfg, slog.Default())

	// Seed the count using the normalized key.
	normalizedKey := "contact:rate:alice@example.com"
	store.setCount(normalizedKey, cfg.RateLimit)

	// Submit with mixed-case email — should still hit the rate limit.
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

// TestService_Submit_BelowRateLimit verifies that submission at count < limit succeeds.
func TestService_Submit_BelowRateLimit(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeKV()
	q := queue.NewDispatcher(nil)
	q.Register(QueueName, func(_ context.Context, _ queue.Job) error { return nil })

	cfg := ServiceConfig{
		RateLimit:  3,
		RateWindow: time.Minute,
		QueueName:  QueueName,
	}
	svc := NewService(repo, store, q, cfg, slog.Default())

	// Seed count one below the limit.
	key := "contact:rate:alice@example.com"
	store.setCount(key, cfg.RateLimit-1)

	input := ContactInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
	}
	if err := svc.Submit(context.Background(), input, "127.0.0.1"); err != nil {
		t.Fatalf("expected success when below rate limit, got: %v", err)
	}
}
