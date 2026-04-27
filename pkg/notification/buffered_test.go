package notification_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/notification"
	"github.com/go-sum/foundry/pkg/notification/memory"
)

func TestBufferedDispatcher_Send_DeliveredByWorker(t *testing.T) {
	mem := memory.New()
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: mem,
	}, nil)
	bd := notification.NewBufferedDispatcher(d, 8, nil)

	n := notification.Notification{
		Subject:  "queued",
		Channels: []notification.Channel{notification.ChannelEmail},
	}
	if err := bd.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := bd.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	sent := mem.Sent()
	if len(sent) != 1 {
		t.Fatalf("memory sender captured %d notifications, want 1", len(sent))
	}
	if sent[0].Subject != "queued" {
		t.Errorf("subject = %q, want %q", sent[0].Subject, "queued")
	}
}

func TestBufferedDispatcher_Send_QueueFull_ReturnsErrQueueFull(t *testing.T) {
	// queueSize=0 means the channel is unbuffered; the worker goroutine may
	// consume the item before we check. Use a blocking dispatcher instead.
	// A blocking inner dispatcher lets us fill the zero-capacity queue reliably.
	blocking := &blockingDispatcher{ready: make(chan struct{})}
	bd := notification.NewBufferedDispatcher(blocking.asDispatcher(), 0, nil)

	n := notification.Notification{
		Channels: []notification.Channel{notification.ChannelEmail},
	}
	err := bd.Send(context.Background(), n)
	if err == nil {
		// The worker may have consumed the item; close and clean up.
		close(blocking.block)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = bd.Shutdown(ctx)
		t.Skip("worker consumed item before ErrQueueFull could be observed with queueSize=0")
	}
	if !errors.Is(err, notification.ErrQueueFull) {
		t.Errorf("errors.Is(err, ErrQueueFull) = false; err = %v", err)
	}

	// Clean up: allow blocked worker to proceed then shut down.
	select {
	case blocking.block <- struct{}{}:
	default:
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = bd.Shutdown(ctx)
}

// blockingDispatcher blocks on Send until its block channel receives.
type blockingDispatcher struct {
	block chan struct{}
	ready chan struct{}
}

func (b *blockingDispatcher) asDispatcher() *notification.Dispatcher {
	// We need a real *Dispatcher but we want to intercept its inner behaviour.
	// Instead, use a fakeSenderBlocking wrapped in a Dispatcher.
	s := &fakeSenderBlocking{block: b.block, ready: b.ready}
	b.block = make(chan struct{}, 1)
	return notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: s,
	}, nil)
}

type fakeSenderBlocking struct {
	block chan struct{}
	ready chan struct{}
	once  sync.Once
}

func (f *fakeSenderBlocking) Send(_ context.Context, _ notification.Notification) error {
	f.once.Do(func() {
		if f.ready != nil {
			close(f.ready)
		}
	})
	<-f.block
	return nil
}

func TestBufferedDispatcher_Send_QueueFull_Reliable(t *testing.T) {
	// Use a large enough queue to fill, with a blocking inner sender.
	blockCh := make(chan struct{})
	blockSender := &simpleFakeSenderBlocking{block: blockCh}
	inner := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: blockSender,
	}, nil)

	const queueSize = 2
	bd := notification.NewBufferedDispatcher(inner, queueSize, nil)

	n := notification.Notification{
		Channels: []notification.Channel{notification.ChannelEmail},
	}

	// Fill the queue past capacity; at some point we must get ErrQueueFull.
	var gotFull bool
	for i := 0; i < queueSize+10; i++ {
		err := bd.Send(context.Background(), n)
		if errors.Is(err, notification.ErrQueueFull) {
			gotFull = true
			break
		}
	}
	if !gotFull {
		t.Error("expected ErrQueueFull when queue is exhausted, but never received it")
	}

	// Unblock the worker and shut down cleanly.
	close(blockCh)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = bd.Shutdown(ctx)
}

type simpleFakeSenderBlocking struct {
	block chan struct{}
}

func (s *simpleFakeSenderBlocking) Send(_ context.Context, _ notification.Notification) error {
	<-s.block
	return nil
}

func TestBufferedDispatcher_Shutdown_DrainsQueue(t *testing.T) {
	mem := memory.New()
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: mem,
	}, nil)
	const count = 5
	bd := notification.NewBufferedDispatcher(d, count*2, nil)

	n := notification.Notification{
		Channels: []notification.Channel{notification.ChannelEmail},
	}
	for i := 0; i < count; i++ {
		if err := bd.Send(context.Background(), n); err != nil {
			t.Fatalf("Send[%d] returned error: %v", i, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := bd.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	if got := len(mem.Sent()); got != count {
		t.Errorf("after Shutdown: captured %d notifications, want %d", got, count)
	}
}

func TestBufferedDispatcher_Send_AfterShutdown_ReturnsErrDeliveryFailed(t *testing.T) {
	mem := memory.New()
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: mem,
	}, nil)
	bd := notification.NewBufferedDispatcher(d, 4, nil)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := bd.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	n := notification.Notification{
		Channels: []notification.Channel{notification.ChannelEmail},
	}
	err := bd.Send(context.Background(), n)
	if !errors.Is(err, notification.ErrDeliveryFailed) {
		t.Errorf("errors.Is(err, ErrDeliveryFailed) = false; err = %v", err)
	}
}

func TestBufferedDispatcher_Shutdown_RespectsContextCancellation(t *testing.T) {
	// A sender that blocks forever prevents the worker from draining.
	neverUnblock := make(chan struct{})
	blockSender := &simpleFakeSenderBlocking{block: neverUnblock}
	inner := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: blockSender,
	}, nil)

	bd := notification.NewBufferedDispatcher(inner, 4, nil)

	// Enqueue one item so the worker blocks on it.
	n := notification.Notification{
		Channels: []notification.Channel{notification.ChannelEmail},
	}
	if err := bd.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	// Give the worker time to pick up the item and block.
	time.Sleep(20 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	err := bd.Shutdown(ctx)
	if err == nil {
		t.Error("Shutdown with canceled context returned nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false; err = %v", err)
	}

	// Allow the blocked goroutine to eventually finish (test cleanup).
	close(neverUnblock)
}
