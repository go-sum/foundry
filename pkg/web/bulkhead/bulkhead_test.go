package bulkhead_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/bulkhead"
)

func TestBulkhead_AcquireAndReleaseWithinCapacity(t *testing.T) {
	b := bulkhead.New("test", 2)

	release1, err := b.Acquire(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on first acquire: %v", err)
	}
	release2, err := b.Acquire(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on second acquire: %v", err)
	}

	release1()
	release2()
}

func TestBulkhead_SaturationReturnsErrTransient(t *testing.T) {
	b := bulkhead.New("test", 1)

	release, err := b.Acquire(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on first acquire: %v", err)
	}
	defer release()

	_, err = b.Acquire(context.Background())
	if !errors.Is(err, web.ErrTransient) {
		t.Fatalf("err = %v, want web.ErrTransient", err)
	}
	if !errors.Is(err, bulkhead.ErrExhausted) {
		t.Fatalf("err = %v, want bulkhead.ErrExhausted", err)
	}
}

func TestBulkhead_AcquireAfterReleaseSucceeds(t *testing.T) {
	b := bulkhead.New("test", 1)

	release, err := b.Acquire(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	release()

	release2, err := b.Acquire(context.Background())
	if err != nil {
		t.Fatalf("unexpected error after release: %v", err)
	}
	release2()
}

func TestBulkhead_ConcurrentAcquireRelease(t *testing.T) {
	b := bulkhead.New("test", 10)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := b.Acquire(context.Background())
			if err == nil {
				release()
			}
		}()
	}
	wg.Wait()
}
