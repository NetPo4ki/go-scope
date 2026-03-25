package scope

import (
	"context"
	"time"
)

// NopObserver is a no-op Observer implementation.
type NopObserver struct{}

func (NopObserver) ScopeCreated(context.Context) {}

func (NopObserver) ScopeCancelled(context.Context, error) {}

func (NopObserver) ScopeJoined(context.Context, time.Duration) {}

func (NopObserver) TaskStarted(context.Context) {}

func (NopObserver) TaskFinished(context.Context, time.Duration, error, bool) {}

type chainedObserver struct {
	observers []Observer
}

// ChainObservers combines multiple observers into one.
// Nil observers are ignored. Returns nil when no observers are provided.
func ChainObservers(observers ...Observer) Observer {
	filtered := make([]Observer, 0, len(observers))
	for _, o := range observers {
		if o != nil {
			filtered = append(filtered, o)
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &chainedObserver{observers: filtered}
	}
}

func (c *chainedObserver) ScopeCreated(ctx context.Context) {
	for _, o := range c.observers {
		o.ScopeCreated(ctx)
	}
}

func (c *chainedObserver) ScopeCancelled(ctx context.Context, cause error) {
	for _, o := range c.observers {
		o.ScopeCancelled(ctx, cause)
	}
}

func (c *chainedObserver) ScopeJoined(ctx context.Context, wait time.Duration) {
	for _, o := range c.observers {
		o.ScopeJoined(ctx, wait)
	}
}

func (c *chainedObserver) TaskStarted(ctx context.Context) {
	for _, o := range c.observers {
		o.TaskStarted(ctx)
	}
}

func (c *chainedObserver) TaskFinished(ctx context.Context, dur time.Duration, err error, panicked bool) {
	for _, o := range c.observers {
		o.TaskFinished(ctx, dur, err, panicked)
	}
}
