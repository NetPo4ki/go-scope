package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/NetPo4ki/go-scope/scope"
)

func TestRecorderOrderWithScope(t *testing.T) {
	t.Parallel()
	rec := NewRecorder()
	s := scope.New(context.Background(), scope.FailFast, scope.WithObserver(rec))
	s.Go(func(_ context.Context) error { return nil })
	if err := s.Wait(); err != nil {
		t.Fatal(err)
	}

	ev := rec.Events()
	if len(ev) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(ev))
	}
	if ev[0].Kind != EventScopeCreated {
		t.Fatalf("first event: %v", ev[0].Kind)
	}
	foundFinish := false
	for _, e := range ev {
		if e.Kind == EventTaskFinished && e.TaskErr == nil && !e.Panicked {
			foundFinish = true
		}
	}
	if !foundFinish {
		t.Fatal("missing successful TaskFinished event")
	}
}

func TestRecorderCancelAndJoin(t *testing.T) {
	t.Parallel()
	rec := NewRecorder()
	s := scope.New(context.Background(), scope.FailFast, scope.WithObserver(rec))
	s.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	s.Cancel(errors.New("stop"))
	_ = s.Wait()

	ev := rec.Events()
	var seenCancel, seenJoin bool
	for _, e := range ev {
		switch e.Kind {
		case EventScopeCancelled:
			seenCancel = true
		case EventScopeJoined:
			seenJoin = true
		}
	}
	if !seenCancel || !seenJoin {
		t.Fatalf("expected cancel and join events, got %#v", ev)
	}
}
