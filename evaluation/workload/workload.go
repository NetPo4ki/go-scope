package workload

import (
	"context"
	"runtime"
	"time"
)

// CPUBound burns CPU for approximately d with cooperative cancellation checks.
// Suitable for measuring scheduling overhead without syscalls.
func CPUBound(ctx context.Context, d time.Duration) error {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		// Check cancellation every ~1000 iterations to keep overhead low
		// while remaining responsive.
		for i := 0; i < 1000; i++ {
			runtime.Gosched()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

// IOBound simulates an I/O wait (network call, disk read) for d with
// cooperative cancellation.
func IOBound(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Nop is a zero-cost task that returns immediately.
// Used to isolate pure framework overhead.
func Nop(_ context.Context) error { return nil }
