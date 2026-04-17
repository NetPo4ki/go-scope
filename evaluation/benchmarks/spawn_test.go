package benchmarks

// B-1: Happy path spawn/wait — measures pure framework overhead.
//
// Three approaches × four scales (N = 1, 10, 100, 1000).
// All tasks do nothing (return nil) to isolate scheduling cost.

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

func BenchmarkSpawnWait_Bare(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithCancel(context.Background())
				var wg sync.WaitGroup
				for j := 0; j < n; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
					}()
				}
				wg.Wait()
				cancel()
				_ = ctx
			}
		})
	}
}

func BenchmarkSpawnWait_Errgroup(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				g, _ := errgroup.WithContext(context.Background())
				for j := 0; j < n; j++ {
					g.Go(func() error { return nil })
				}
				_ = g.Wait()
			}
		})
	}
}

func BenchmarkSpawnWait_Scope(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s := scope.New(context.Background(), scope.FailFast)
				for j := 0; j < n; j++ {
					s.Go(func(_ context.Context) error { return nil })
				}
				_ = s.Wait()
			}
		})
	}
}
