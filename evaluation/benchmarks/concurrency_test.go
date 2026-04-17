package benchmarks

// B-6: MaxConcurrency — measures the overhead of semaphore-bounded execution.
//
// N = 100 tasks with concurrency limit = 8.
// Bare Go baseline uses a buffered channel as semaphore.

import (
	"context"
	"sync"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

const (
	concN     = 100
	concLimit = 8
)

func BenchmarkConcurrency_Bare(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		sem := make(chan struct{}, concLimit)
		var wg sync.WaitGroup
		for j := 0; j < concN; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					return
				}
			}()
		}
		wg.Wait()
		cancel()
	}
}

func BenchmarkConcurrency_Errgroup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g, _ := errgroup.WithContext(context.Background())
		g.SetLimit(concLimit)
		for j := 0; j < concN; j++ {
			g.Go(func() error { return nil })
		}
		_ = g.Wait()
	}
}

func BenchmarkConcurrency_Scope(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := scope.New(context.Background(), scope.FailFast, scope.WithMaxConcurrency(concLimit))
		for j := 0; j < concN; j++ {
			s.Go(func(_ context.Context) error { return nil })
		}
		_ = s.Wait()
	}
}
