package scope

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recObserver struct {
	created   int
	cancelled int
	joined    int
	started   int
	finished  int
}

func (o *recObserver) ScopeCreated(context.Context) { o.created++ }

func (o *recObserver) ScopeCancelled(context.Context, error) { o.cancelled++ }

func (o *recObserver) ScopeJoined(context.Context, time.Duration) { o.joined++ }

func (o *recObserver) TaskStarted(context.Context) { o.started++ }

func (o *recObserver) TaskFinished(context.Context, time.Duration, error, bool) { o.finished++ }

func TestChainObserversNilHandling(t *testing.T) {
	t.Parallel()
	if got := ChainObservers(nil, nil); got != nil {
		t.Fatal("expected nil chain when all observers are nil")
	}
}

func TestChainObserversForwardsEvents(t *testing.T) {
	t.Parallel()
	o1 := &recObserver{}
	o2 := &recObserver{}
	obs := ChainObservers(o1, nil, o2)

	ctx := context.Background()
	obs.ScopeCreated(ctx)
	obs.ScopeCancelled(ctx, errors.New("x"))
	obs.ScopeJoined(ctx, time.Millisecond)
	obs.TaskStarted(ctx)
	obs.TaskFinished(ctx, time.Millisecond, nil, false)

	if o1.created != 1 || o2.created != 1 {
		t.Fatalf("expected created forwarded to both, got %d and %d", o1.created, o2.created)
	}
	if o1.cancelled != 1 || o2.cancelled != 1 {
		t.Fatalf("expected cancel forwarded to both, got %d and %d", o1.cancelled, o2.cancelled)
	}
	if o1.joined != 1 || o2.joined != 1 {
		t.Fatalf("expected joined forwarded to both, got %d and %d", o1.joined, o2.joined)
	}
	if o1.started != 1 || o2.started != 1 || o1.finished != 1 || o2.finished != 1 {
		t.Fatalf("expected task events forwarded to both, got start(%d,%d) finish(%d,%d)",
			o1.started, o2.started, o1.finished, o2.finished)
	}
}

