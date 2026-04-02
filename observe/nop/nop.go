// Package nop provides a no-op [scope.Observer] for disabling instrumentation.
package nop

import (
	"context"
	"time"

	"github.com/NetPo4ki/go-scope/scope"
)

// Observer implements [scope.Observer] with empty methods.
type Observer struct{}

// New returns a new no-op observer.
func New() scope.Observer { return &Observer{} }

// ScopeCreated implements [scope.Observer].
func (*Observer) ScopeCreated(context.Context) {}

// ScopeCancelled implements [scope.Observer].
func (*Observer) ScopeCancelled(context.Context, error) {}

// ScopeJoined implements [scope.Observer].
func (*Observer) ScopeJoined(context.Context, time.Duration) {}

// TaskStarted implements [scope.Observer].
func (*Observer) TaskStarted(context.Context) {}

// TaskFinished implements [scope.Observer].
func (*Observer) TaskFinished(context.Context, time.Duration, error, bool) {}
