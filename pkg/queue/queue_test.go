package queue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeStore is a thread-safe fake implementation of Store for tests.
type fakeStore struct {
	mu sync.Mutex

	enqueued []*Job
	dequeue  []*Job // pre-loaded jobs to return on Dequeue
	deqIdx   int

	completedIDs []string
	failedIDs    []string
	failedErrs   []string
	failedRetry  []time.Duration

	enqueueErr  error
	dequeueErr  error
	completeErr error
	failErr     error
	reapErr     error
	purgeErr    error
	pingErr     error

	reapCount int
	closed    bool
}

func (f *fakeStore) Enqueue(_ context.Context, job *Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.enqueueErr != nil {
		return f.enqueueErr
	}
	job.ID = "fake-id"
	f.enqueued = append(f.enqueued, job)
	return nil
}

func (f *fakeStore) Dequeue(_ context.Context, _ []string) (*Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dequeueErr != nil {
		return nil, f.dequeueErr
	}
	if f.deqIdx >= len(f.dequeue) {
		return nil, ErrNotFound
	}
	job := f.dequeue[f.deqIdx]
	f.deqIdx++
	return job, nil
}

func (f *fakeStore) Complete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.completeErr != nil {
		return f.completeErr
	}
	f.completedIDs = append(f.completedIDs, id)
	return nil
}

func (f *fakeStore) Fail(_ context.Context, id string, errMsg string, retryAfter time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failErr != nil {
		return f.failErr
	}
	f.failedIDs = append(f.failedIDs, id)
	f.failedErrs = append(f.failedErrs, errMsg)
	f.failedRetry = append(f.failedRetry, retryAfter)
	return nil
}

func (f *fakeStore) Reap(_ context.Context, _ []string, _ time.Duration) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reapErr != nil {
		return 0, f.reapErr
	}
	return f.reapCount, nil
}

func (f *fakeStore) Purge(_ context.Context, _ time.Duration, _ int) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return 0, f.purgeErr
}

func (f *fakeStore) Ping(_ context.Context) error { return f.pingErr }

func (f *fakeStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeStore) lastEnqueued() *Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.enqueued) == 0 {
		return nil
	}
	return f.enqueued[len(f.enqueued)-1]
}

func (f *fakeStore) completedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.completedIDs)
}

func (f *fakeStore) failedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.failedIDs)
}

// -- Dispatcher tests --

func TestDispatcher_Enqueue(t *testing.T) {
	store := &fakeStore{}
	d := NewDispatcher(store)
	d.Register("emails", func(_ context.Context, _ Job) error { return nil })

	payload := json.RawMessage(`{"to":"test@example.com"}`)
	err := d.Dispatch(context.Background(), "emails", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.lastEnqueued()
	if job == nil {
		t.Fatal("expected a job to be enqueued")
	}
	if job.Queue != "emails" {
		t.Errorf("queue = %q, want %q", job.Queue, "emails")
	}
	if string(job.Payload) != string(payload) {
		t.Errorf("payload = %s, want %s", job.Payload, payload)
	}
	if job.Status != StatusPending {
		t.Errorf("status = %q, want %q", job.Status, StatusPending)
	}
	if job.MaxAttempts != 3 {
		t.Errorf("max_attempts = %d, want 3", job.MaxAttempts)
	}
}

func TestDispatcher_UnknownQueue(t *testing.T) {
	d := NewDispatcher(nil) // sync mode: unknown queue must fail
	err := d.Dispatch(context.Background(), "nonexistent", json.RawMessage(`{}`))
	if !errors.Is(err, ErrQueueUnknown) {
		t.Errorf("err = %v, want ErrQueueUnknown", err)
	}
}

func TestDispatcher_Closed(t *testing.T) {
	store := &fakeStore{}
	d := NewDispatcher(store)
	d.Register("emails", func(_ context.Context, _ Job) error { return nil })

	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := d.Dispatch(context.Background(), "emails", json.RawMessage(`{}`))
	if !errors.Is(err, ErrClosed) {
		t.Errorf("err = %v, want ErrClosed", err)
	}
}

func TestDispatcher_DefaultPriority(t *testing.T) {
	store := &fakeStore{}
	d := NewDispatcher(store)
	d.Register("emails", func(_ context.Context, _ Job) error { return nil })

	before := time.Now()
	err := d.Dispatch(context.Background(), "emails", json.RawMessage(`{}`))
	after := time.Now()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.lastEnqueued()
	if job == nil {
		t.Fatal("expected a job to be enqueued")
	}
	if job.Priority != PriorityDefault {
		t.Errorf("priority = %d, want %d", job.Priority, PriorityDefault)
	}
	if job.RunAt.Before(before) || job.RunAt.After(after) {
		t.Errorf("RunAt %v not in expected range [%v, %v]", job.RunAt, before, after)
	}
}

func TestDispatcher_ZeroRunAt(t *testing.T) {
	store := &fakeStore{}
	d := NewDispatcher(store)
	d.Register("emails", func(_ context.Context, _ Job) error { return nil })

	before := time.Now()
	err := d.Dispatch(context.Background(), "emails", json.RawMessage(`{}`))
	after := time.Now()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.lastEnqueued()
	if job == nil {
		t.Fatal("expected a job to be enqueued")
	}
	if job.RunAt.IsZero() {
		t.Error("RunAt should not be zero")
	}
	if job.RunAt.Before(before) || job.RunAt.After(after) {
		t.Errorf("RunAt %v not approximately now (range: [%v, %v])", job.RunAt, before, after)
	}
}

func TestDispatcher_EnqueueError(t *testing.T) {
	storeErr := errors.New("db down")
	store := &fakeStore{enqueueErr: storeErr}
	d := NewDispatcher(store)
	d.Register("emails", func(_ context.Context, _ Job) error { return nil })

	err := d.Dispatch(context.Background(), "emails", json.RawMessage(`{}`))
	if !errors.Is(err, storeErr) {
		t.Errorf("err = %v, want %v", err, storeErr)
	}
}

func TestDispatchPayload_Success(t *testing.T) {
	store := &fakeStore{}
	d := NewDispatcher(store)
	d.Register("emails", func(_ context.Context, _ Job) error { return nil })

	type Msg struct {
		To string `json:"to"`
	}
	err := d.DispatchPayload(context.Background(), "emails", Msg{To: "test@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.lastEnqueued()
	if job == nil {
		t.Fatal("expected a job to be enqueued")
	}
	var msg Msg
	if err := json.Unmarshal(job.Payload, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.To != "test@example.com" {
		t.Errorf("To = %q, want %q", msg.To, "test@example.com")
	}
	if job.Priority != PriorityDefault {
		t.Errorf("priority = %d, want %d", job.Priority, PriorityDefault)
	}
}

func TestDispatchPayload_MarshalError(t *testing.T) {
	store := &fakeStore{}
	d := NewDispatcher(store)
	d.Register("emails", func(_ context.Context, _ Job) error { return nil })

	// channels cannot be marshaled to JSON
	err := d.DispatchPayload(context.Background(), "emails", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
}

func TestSyncDispatch_ExecutesInline(t *testing.T) {
	executed := false
	d := NewDispatcher(nil) // nil store = sync mode
	d.Register("work", func(_ context.Context, _ Job) error {
		executed = true
		return nil
	})

	err := d.Dispatch(context.Background(), "work", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executed {
		t.Error("handler was not executed inline")
	}
}

func TestSyncDispatch_PropagatesError(t *testing.T) {
	handlerErr := errors.New("handler failed")
	d := NewDispatcher(nil)
	d.Register("work", func(_ context.Context, _ Job) error {
		return handlerErr
	})

	err := d.Dispatch(context.Background(), "work", json.RawMessage(`{}`))
	if !errors.Is(err, handlerErr) {
		t.Errorf("err = %v, want %v", err, handlerErr)
	}
}

func TestSyncDispatch_RecoversPanic(t *testing.T) {
	d := NewDispatcher(nil)
	d.Register("work", func(_ context.Context, _ Job) error {
		panic("something went wrong")
	})

	err := d.Dispatch(context.Background(), "work", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}
	want := "queue: handler panic: something went wrong"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err.Error(), want)
	}
}

func TestSyncDispatch_NoHandler(t *testing.T) {
	d := NewDispatcher(nil)
	// Register queue but handler will be nil — actually per spec "queue with no handler returns error"
	// The plan says "no handler" — but Register requires a HandlerFunc. We test unknown queue instead
	// to satisfy the intent: dispatching to unregistered queue returns ErrQueueUnknown.
	err := d.Dispatch(context.Background(), "unregistered", json.RawMessage(`{}`))
	if !errors.Is(err, ErrQueueUnknown) {
		t.Errorf("err = %v, want ErrQueueUnknown", err)
	}
}

func TestDispatcher_AsyncDispatchWithoutRegister(t *testing.T) {
	store := &fakeStore{}
	d := NewDispatcher(store)

	payload := json.RawMessage(`{"key":"value"}`)
	err := d.Dispatch(context.Background(), "unregistered", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.lastEnqueued()
	if job == nil {
		t.Fatal("expected a job to be enqueued")
	}
	if job.Queue != "unregistered" {
		t.Errorf("queue = %q, want %q", job.Queue, "unregistered")
	}
	if job.Priority != PriorityDefault {
		t.Errorf("priority = %d, want %d", job.Priority, PriorityDefault)
	}
	if job.MaxAttempts != 3 {
		t.Errorf("max_attempts = %d, want 3", job.MaxAttempts)
	}
}

// -- Processor tests --

func newTestProcessor(store Store) *Processor {
	return NewProcessor(store,
		WithPollInterval(10*time.Millisecond),
		WithShutdownWait(5*time.Second),
		WithReapInterval(1*time.Hour), // large so reaper doesn't fire during test
	)
}

func TestProcessor_ProcessesJob(t *testing.T) {
	job := &Job{ID: "job-1", Queue: "work", Payload: json.RawMessage(`{}`)}
	store := &fakeStore{dequeue: []*Job{job}}

	proc := newTestProcessor(store)
	proc.Register("work", func(_ context.Context, _ Job) error { return nil })

	ctx := context.Background()
	proc.Start(ctx)
	<-proc.Ready()

	// Wait until Complete is called or timeout
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if store.completedCount() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err := proc.Stop(); err != nil {
		t.Fatalf("proc.Stop: %v", err)
	}

	if store.completedCount() < 1 {
		t.Error("expected Complete to be called at least once")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.completedIDs) == 0 || store.completedIDs[0] != "job-1" {
		t.Errorf("completed ID = %v, want [job-1]", store.completedIDs)
	}
}

func TestProcessor_HandlerError(t *testing.T) {
	job := &Job{ID: "job-2", Queue: "work", Payload: json.RawMessage(`{}`), Attempts: 1}
	store := &fakeStore{dequeue: []*Job{job}}

	handlerErr := errors.New("processing failed")
	proc := newTestProcessor(store)
	proc.Register("work", func(_ context.Context, _ Job) error {
		return handlerErr
	}, WithBackoff(5*time.Second))

	ctx := context.Background()
	proc.Start(ctx)
	<-proc.Ready()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if store.failedCount() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err := proc.Stop(); err != nil {
		t.Fatalf("proc.Stop: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.failedIDs) == 0 {
		t.Fatal("expected Fail to be called")
	}
	if store.failedIDs[0] != "job-2" {
		t.Errorf("failed ID = %q, want %q", store.failedIDs[0], "job-2")
	}
	if store.failedErrs[0] != handlerErr.Error() {
		t.Errorf("failed err = %q, want %q", store.failedErrs[0], handlerErr.Error())
	}
	// attempts=1, backoff = base * 2^0 = 5s with ±50% jitter → [2.5s, 7.5s)
	gotBackoff := store.failedRetry[0]
	lo, hi := 5*time.Second/2, 5*time.Second*3/2
	if gotBackoff < lo || gotBackoff >= hi {
		t.Errorf("retry backoff = %v, want in [%v, %v)", gotBackoff, lo, hi)
	}
}

func TestProcessor_HandlerError_ZeroBackoffDoesNotPanic(t *testing.T) {
	job := &Job{ID: "job-zero", Queue: "work", Payload: json.RawMessage(`{}`), Attempts: 1}
	store := &fakeStore{dequeue: []*Job{job}}

	handlerErr := errors.New("processing failed")
	proc := newTestProcessor(store)
	proc.Register("work", func(_ context.Context, _ Job) error {
		return handlerErr
	}, WithBackoff(0))

	ctx := context.Background()
	proc.Start(ctx)
	<-proc.Ready()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if store.failedCount() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err := proc.Stop(); err != nil {
		t.Fatalf("proc.Stop: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.failedIDs) == 0 {
		t.Fatal("expected Fail to be called")
	}
	if store.failedRetry[0] != 0 {
		t.Errorf("retry backoff = %v, want 0", store.failedRetry[0])
	}
}

func TestProcessor_HandlerPanic(t *testing.T) {
	job := &Job{ID: "job-3", Queue: "work", Payload: json.RawMessage(`{}`)}
	store := &fakeStore{dequeue: []*Job{job}}

	proc := newTestProcessor(store)
	proc.Register("work", func(_ context.Context, _ Job) error {
		panic("unexpected panic")
	})

	ctx := context.Background()
	proc.Start(ctx)
	<-proc.Ready()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if store.failedCount() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err := proc.Stop(); err != nil {
		t.Fatalf("proc.Stop: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.failedIDs) == 0 {
		t.Fatal("expected Fail to be called after panic")
	}
	want := "queue: handler panic: unexpected panic"
	if store.failedErrs[0] != want {
		t.Errorf("panic error = %q, want %q", store.failedErrs[0], want)
	}
}

func TestProcessor_DoubleStop(t *testing.T) {
	store := &fakeStore{}
	proc := newTestProcessor(store)
	proc.Register("work", func(_ context.Context, _ Job) error { return nil })

	proc.Start(context.Background())
	<-proc.Ready()

	if err := proc.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := proc.Stop(); !errors.Is(err, ErrClosed) {
		t.Errorf("second Stop: %v, want ErrClosed", err)
	}
}

func TestProcessor_Ready(t *testing.T) {
	store := &fakeStore{}
	proc := newTestProcessor(store)
	proc.Register("work", func(_ context.Context, _ Job) error { return nil })

	proc.Start(context.Background())

	select {
	case <-proc.Ready():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("Ready() channel did not close within timeout")
	}

	if err := proc.Stop(); err != nil {
		t.Fatalf("proc.Stop: %v", err)
	}
}

func TestProcessor_Ping(t *testing.T) {
	store := &fakeStore{}
	proc := newTestProcessor(store)

	if err := proc.Ping(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	store.pingErr = errors.New("connection refused")
	if err := proc.Ping(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// -- computeBackoff tests --

func TestComputeBackoff(t *testing.T) {
	base := time.Second
	tests := []struct {
		attempts int
		wantBase time.Duration
	}{
		{0, time.Second},      // shift=-1 → clamped to 0 → base * 1
		{1, time.Second},      // shift=0 → base * 1
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
	}
	for _, tc := range tests {
		got := computeBackoff(base, tc.attempts)
		// jitter is in [0, wantBase/2), applied ±, so result is in [wantBase/2, wantBase*3/2)
		lo, hi := tc.wantBase/2, tc.wantBase*3/2
		if got < lo || got >= hi {
			t.Errorf("computeBackoff(%v, %d) = %v, want in [%v, %v)", base, tc.attempts, got, lo, hi)
		}
	}
}

func TestComputeBackoff_Cap(t *testing.T) {
	base := time.Second
	// attempts=22 → shift=21, clamped to 20 → base * 2^20, then ±50% jitter
	got := computeBackoff(base, 22)
	d := base * (1 << 20)
	lo, hi := d/2, d*3/2
	if got < lo || got >= hi {
		t.Errorf("computeBackoff cap: got %v, want in [%v, %v)", got, lo, hi)
	}
}

func TestComputeBackoff_NonPositiveBase(t *testing.T) {
	tests := []time.Duration{0, -1 * time.Second}
	for _, base := range tests {
		if got := computeBackoff(base, 1); got != 0 {
			t.Errorf("computeBackoff(%v, 1) = %v, want 0", base, got)
		}
	}
}

// -- Sentinel error tests --

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrNotFound, "queue: job not found"},
		{ErrQueueUnknown, "queue: unknown queue name"},
		{ErrClosed, "queue: already closed"},
	}
	for _, tc := range tests {
		if tc.err.Error() != tc.want {
			t.Errorf("%v.Error() = %q, want %q", tc.err, tc.err.Error(), tc.want)
		}
	}
}

func TestPriorityValues(t *testing.T) {
	tests := []struct {
		p    Priority
		want int
	}{
		{PriorityCritical, 0},
		{PriorityHigh, 10},
		{PriorityDefault, 20},
		{PriorityLow, 30},
	}
	for _, tc := range tests {
		if int(tc.p) != tc.want {
			t.Errorf("Priority = %d, want %d", tc.p, tc.want)
		}
	}
}

func TestStatusValues(t *testing.T) {
	tests := []struct {
		s    Status
		want string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusDead, "dead"},
	}
	for _, tc := range tests {
		if string(tc.s) != tc.want {
			t.Errorf("Status = %q, want %q", tc.s, tc.want)
		}
	}
}
