package scope

import (
	"context"
	"time"
)

// deriveContext applies deadline/timeout options consistently.
//
// Priority: explicit deadline, then timeout, then plain cancelable child context.
// Nil parent is treated as context.Background().
func deriveContext(parent context.Context, deadline time.Time, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	switch {
	case !deadline.IsZero():
		return context.WithDeadline(parent, deadline)
	case timeout > 0:
		return context.WithTimeout(parent, timeout)
	default:
		return context.WithCancel(parent)
	}
}
