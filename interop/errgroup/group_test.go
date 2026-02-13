package errgroup

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithContextHappy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	g, gctx := WithContext(ctx)
	_ = gctx
	g.Go(func() error { return nil })
	g.Go(func() error { time.Sleep(10 * time.Millisecond); return nil })
	if err := g.Wait(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithContextErrorCancels(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	g, gctx := WithContext(ctx)
	done := make(chan struct{})
	g.Go(func() error { return errors.New("boom") })
	g.Go(func() error {
		select {
		case <-gctx.Done():
			close(done)
			return nil
		case <-time.After(250 * time.Millisecond):
			t.Fatal("expected cancel propagation")
			return nil
		}
	})
	if err := g.Wait(); err == nil {
		t.Fatal("expected error")
	}
	select {
	case <-done:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("ctx was not canceled")
	}
}

func TestWithContextParentDeadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	g, gctx := WithContext(ctx)
	g.Go(func() error {
		// cooperative task: observe context cancellation
		<-gctx.Done()
		return gctx.Err()
	})
	err := g.Wait()
	if err == nil {
		t.Fatal("expected deadline error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestWithContextParentCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	g, gctx := WithContext(ctx)
	g.Go(func() error {
		// cooperative task: observe context cancellation
		<-gctx.Done()
		return gctx.Err()
	})
	cancel()
	err := g.Wait()
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
