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

func (*Nop) ScopeCreated(context.Context)                             {}
func (*Nop) ScopeCancelled(context.Context, error)                    {}
func (*Nop) ScopeJoined(context.Context, time.Duration)               {}
func (*Nop) TaskStarted(context.Context)                              {}
func (*Nop) TaskFinished(context.Context, time.Duration, error, bool) {}
