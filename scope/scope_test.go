package scope

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestGoWaitSuccess(t *testing.T) {
	t.Parallel()
	s := New(context.Background(), FailFast)
	done := atomic.Int32{}
	s.Go(func(_ context.Context) error {
		done.Add(1)
		return nil
	})
	if err := s.Wait(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := done.Load(); got != 1 {
		t.Fatalf("expected task to run once, got %d", got)
	}
}

func TestCancelIdempotentMultiWait(t *testing.T) {
	t.Parallel()
	s := New(context.Background(), FailFast)
	s.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	s.Cancel(errors.New("stop"))
	s.Cancel(nil)
	err1 := s.Wait()
	err2 := s.Wait()
	if err1 == nil || err2 == nil {
		t.Fatalf("expected non-nil error from Wait after cancel, got (%v, %v)", err1, err2)
	}
	if err1.Error() != err2.Error() {
		t.Fatalf("Wait should return same error; got %v vs %v", err1, err2)
	}
}

func TestFailFastCancelsSiblings(t *testing.T) {
	t.Parallel()
	s := New(context.Background(), FailFast)
	blocked := make(chan struct{})

	s.Go(func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			t.Fatal("sibling was not cancelled by fail-fast")
			return nil
		case <-ctx.Done():
			close(blocked)
			return ctx.Err()
		}
	})
	s.Go(func(_ context.Context) error {
		time.Sleep(30 * time.Millisecond)
		return errors.New("boom")
	})
	if err := s.Wait(); err == nil {
		t.Fatal("expected error from fail-fast scope")
	}
	select {
	case <-blocked:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("sibling did not observe cancellation in time")
	}
}

func TestSupervisorDoesNotCancelSiblings(t *testing.T) {
	t.Parallel()
	s := New(context.Background(), Supervisor)
	done := make(chan struct{})
	s.Go(func(_ context.Context) error {
		time.Sleep(40 * time.Millisecond)
		close(done)
		return nil
	})
	s.Go(func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return errors.New("err")
	})
	if err := s.Wait(); err == nil {
		t.Fatal("expected non-nil error from supervisor Wait")
	}
	select {
	case <-done:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("sibling should not be cancelled under Supervisor policy")
	}
}

func TestPanicAsErrorConverted(t *testing.T) {
	t.Parallel()
	s := New(context.Background(), FailFast, WithPanicAsError(true))
	s.Go(func(ctx context.Context) error {
		panic("panic-value")
	})
	if err := s.Wait(); err == nil || err.Error() == "panic-value" {
		t.Fatalf("expected converted panic error, got %v", err)
	}
}

func TestChildCancellation(t *testing.T) {
	t.Parallel()
	parent := New(context.Background(), FailFast)
	child := parent.Child(FailFast)
	cancelObserved := make(chan struct{})
	child.Go(func(ctx context.Context) error {
		<-ctx.Done()
		close(cancelObserved)
		return ctx.Err()
	})
	parent.Cancel(errors.New("stop"))
	_ = parent.Wait()
	select {
	case <-cancelObserved:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("child did not observe parent's cancellation")
	}
}

type countObserver struct {
	started  atomic.Int64
	finished atomic.Int64
	joined   atomic.Int64
	cancel   atomic.Int64
}

func (o *countObserver) ScopeCreated(_ context.Context)                 {}
func (o *countObserver) ScopeCancelled(_ context.Context, _ error)      { o.cancel.Add(1) }
func (o *countObserver) ScopeJoined(_ context.Context, _ time.Duration) { o.joined.Add(1) }
func (o *countObserver) TaskStarted(_ context.Context)                  { o.started.Add(1) }
func (o *countObserver) TaskFinished(_ context.Context, _ time.Duration, _ error, _ bool) {
	o.finished.Add(1)
}

func TestObserverHooks(t *testing.T) {
	t.Parallel()
	obs := &countObserver{}
	s := New(context.Background(), FailFast, WithObserver(obs))
	s.Go(func(_ context.Context) error { return nil })
	s.Go(func(_ context.Context) error { return nil })
	if err := s.Wait(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.started.Load() != 2 || obs.finished.Load() != 2 || obs.joined.Load() != 1 {
		t.Fatalf("unexpected observer counts: started=%d finished=%d joined=%d",
			obs.started.Load(), obs.finished.Load(), obs.joined.Load())
	}
}
