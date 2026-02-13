package otel

import (
	"context"
	"time"
)

// Nop is a no-op implementation of the scope.Observer interface.
// It serves as a placeholder for an OpenTelemetry-backed observer without adding dependencies.
type Nop struct{}

// NewNop returns a no-op observer.
func NewNop() *Nop { return &Nop{} }

// ScopeCreated is a no-op.
func (*Nop) ScopeCreated(context.Context) {}

// ScopeCancelled is a no-op.
func (*Nop) ScopeCancelled(context.Context, error) {}

// ScopeJoined is a no-op.
func (*Nop) ScopeJoined(context.Context, time.Duration) {}

// TaskStarted is a no-op.
func (*Nop) TaskStarted(context.Context) {}

// TaskFinished is a no-op.
func (*Nop) TaskFinished(context.Context, time.Duration, error, bool) {}
