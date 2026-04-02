package macro

import (
	"context"
	"fmt"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// Simulated sub-request work (cooperative cancellation).
func work(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func BenchmarkFanout_ScopeFailFast(b *testing.B) {
	const (
		nOK     = 8
		latency = 50 * time.Microsecond
	)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := scope.New(context.Background(), scope.FailFast)
		for k := 0; k < nOK; k++ {
			s.Go(func(ctx context.Context) error { return work(ctx, latency) })
		}
		s.Go(func(context.Context) error { return fmt.Errorf("injected") })
		_ = s.Wait()
	}
}

func BenchmarkFanout_Errgroup(b *testing.B) {
	const (
		nOK     = 8
		latency = 50 * time.Microsecond
	)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g, ctx := errgroup.WithContext(context.Background())
		for k := 0; k < nOK; k++ {
			g.Go(func() error { return work(ctx, latency) })
		}
		g.Go(func() error { return fmt.Errorf("injected") })
		_ = g.Wait()
	}
}
