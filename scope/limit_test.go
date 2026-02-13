package scope

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestMaxConcurrencyBound(t *testing.T) {
	t.Parallel()
	const N = 8
	const M = 50
	s := New(context.Background(), Supervisor, WithMaxConcurrency(N))
	var cur, maxSeen atomic.Int64
	block := make(chan struct{})
	for i := 0; i < M; i++ {
		s.Go(func(ctx context.Context) error {
			c := cur.Add(1)
			for {
				if m := maxSeen.Load(); c > m {
					maxSeen.CompareAndSwap(m, c)
				}
				select {
				case <-block:
					cur.Add(-1)
					return nil
				case <-ctx.Done():
					cur.Add(-1)
					return ctx.Err()
				case <-time.After(1 * time.Millisecond):
				}
			}
		})
	}
	time.Sleep(50 * time.Millisecond)
	close(block)
	_ = s.Wait()
	if observed := int(maxSeen.Load()); observed > N {
		t.Fatalf("observed concurrency %d exceeds limit %d", observed, N)
	}
}

func TestLimiterAcquireRespectsCancel(t *testing.T) {
	t.Parallel()
	s := New(context.Background(), FailFast, WithMaxConcurrency(1))
	block := make(chan struct{})
	s.Go(func(_ context.Context) error {
		<-block
		return nil
	})
	// start a second task that will be blocked on Acquire
	s.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	// Give goroutine time to attempt Acquire
	time.Sleep(10 * time.Millisecond)
	// Measure cancellation responsiveness
	start := time.Now()
	s.Cancel(context.Canceled)
	// Release the first task so Wait() can finish promptly
	close(block)
	_ = s.Wait()
	elapsed := time.Since(start)
	if elapsed > 300*time.Millisecond {
		t.Fatalf("expected quick abort on cancel, got %v", elapsed)
	}
}

func TestChildMaxConcurrencyBound(t *testing.T) {
	t.Parallel()
	parent := New(context.Background(), Supervisor)
	child := parent.Child(Supervisor, WithMaxConcurrency(1))
	var cur, maxSeen atomic.Int64
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})

	child.Go(func(_ context.Context) error {
		c := cur.Add(1)
		for {
			if m := maxSeen.Load(); c > m {
				maxSeen.CompareAndSwap(m, c)
			}
			select {
			case <-ch1:
				cur.Add(-1)
				return nil
			case <-time.After(1 * time.Millisecond):
			}
		}
	})
	child.Go(func(_ context.Context) error {
		c := cur.Add(1)
		for {
			if m := maxSeen.Load(); c > m {
				maxSeen.CompareAndSwap(m, c)
			}
			select {
			case <-ch2:
				cur.Add(-1)
				return nil
			case <-time.After(1 * time.Millisecond):
			}
		}
	})
	// Let first task start; second should be queued by limiter.
	time.Sleep(20 * time.Millisecond)
	if observed := int(maxSeen.Load()); observed > 1 {
		t.Fatalf("child observed concurrency %d exceeds limit 1", observed)
	}
	// Release first, then second.
	close(ch1)
	time.Sleep(20 * time.Millisecond)
	close(ch2)
	_ = child.Wait()
	_ = parent.Wait()
}
