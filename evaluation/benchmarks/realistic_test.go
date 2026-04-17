package benchmarks

// B-7: Realistic workload — tasks with I/O-like latency.
//
// Shows that scope's framework overhead (~300-600ns) becomes invisible
// when tasks perform real work (1ms+ I/O). This is the "does it matter
// in practice?" benchmark.

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

func ioWork(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func BenchmarkRealistic_Bare(b *testing.B) {
	for _, lat := range []time.Duration{100 * time.Microsecond, time.Millisecond} {
		for _, n := range []int{10, 100} {
			b.Run(fmt.Sprintf("lat=%v/N=%d", lat, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					ctx := context.Background()
					var wg sync.WaitGroup
					for j := 0; j < n; j++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							_ = ioWork(ctx, lat)
						}()
					}
					wg.Wait()
				}
			})
		}
	}
}

func BenchmarkRealistic_Errgroup(b *testing.B) {
	for _, lat := range []time.Duration{100 * time.Microsecond, time.Millisecond} {
		for _, n := range []int{10, 100} {
			b.Run(fmt.Sprintf("lat=%v/N=%d", lat, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					g, ctx := errgroup.WithContext(context.Background())
					for j := 0; j < n; j++ {
						g.Go(func() error {
							return ioWork(ctx, lat)
						})
					}
					_ = g.Wait()
				}
			})
		}
	}
}

func BenchmarkRealistic_Scope(b *testing.B) {
	for _, lat := range []time.Duration{100 * time.Microsecond, time.Millisecond} {
		for _, n := range []int{10, 100} {
			b.Run(fmt.Sprintf("lat=%v/N=%d", lat, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					s := scope.New(context.Background(), scope.FailFast)
					for j := 0; j < n; j++ {
						s.Go(func(ctx context.Context) error {
							return ioWork(ctx, lat)
						})
					}
					_ = s.Wait()
				}
			})
		}
	}
}
