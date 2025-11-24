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
	var cur, max atomic.Int64
	block := make(chan struct{})
	for i := 0; i < M; i++ {
		s.Go(func(ctx context.Context) error {
			c := cur.Add(1)
			for {
				if m := max.Load(); c > m {
					max.CompareAndSwap(m, c)
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
	if observed := int(max.Load()); observed > N {
		t.Fatalf("observed concurrency %d exceeds limit %d", observed, N)
	}
}

func TestLimiterAcquireRespectsCancel(t *testing.T) {
	t.Parallel()
	s := New(context.Background(), FailFast, WithMaxConcurrency(1))
	block := make(chan struct{})
	s.Go(func(ctx context.Context) error {
		<-block
		return nil
	})
	start := time.Now()
	done := make(chan struct{})
	s.Go(func(ctx context.Context) error {
		close(done)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			t.Fatal("acquire did not respect cancellation")
			return nil
		}
	})
	<-done
	s.Cancel(context.Canceled)
	_ = s.Wait()
	elapsed := time.Since(start)
	if elapsed > 300*time.Millisecond {
		t.Fatalf("expected quick abort on cancel, got %v", elapsed)
	}
}
