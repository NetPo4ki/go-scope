package benchmarks

// B-3: Supervisor error aggregation.
//
// N tasks, K = N/4 return errors. Measures the cost of collecting all errors
// vs the bare-Go pattern (mutex + []error) and errgroup (first-error-only).

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

var errBench = errors.New("bench")

func BenchmarkSupervisor_Bare(b *testing.B) {
	for _, n := range []int{10, 100} {
		k := n / 4
		b.Run(fmt.Sprintf("N=%d_K=%d", n, k), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var (
					wg   sync.WaitGroup
					mu   sync.Mutex
					errs []error
				)
				for j := 0; j < n; j++ {
					fail := j < k
					wg.Add(1)
					go func() {
						defer wg.Done()
						if fail {
							mu.Lock()
							errs = append(errs, errBench)
							mu.Unlock()
						}
					}()
				}
				wg.Wait()
				_ = errors.Join(errs...)
			}
		})
	}
}

func BenchmarkSupervisor_Errgroup(b *testing.B) {
	for _, n := range []int{10, 100} {
		k := n / 4
		b.Run(fmt.Sprintf("N=%d_K=%d", n, k), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				g, _ := errgroup.WithContext(context.Background())
				for j := 0; j < n; j++ {
					fail := j < k
					g.Go(func() error {
						if fail {
							return errBench
						}
						return nil
					})
				}
				// errgroup only captures first error — cannot aggregate.
				_ = g.Wait()
			}
		})
	}
}

func BenchmarkSupervisor_Scope(b *testing.B) {
	for _, n := range []int{10, 100} {
		k := n / 4
		b.Run(fmt.Sprintf("N=%d_K=%d", n, k), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s := scope.New(context.Background(), scope.Supervisor)
				for j := 0; j < n; j++ {
					fail := j < k
					s.Go(func(_ context.Context) error {
						if fail {
							return errBench
						}
						return nil
					})
				}
				_ = s.Wait()
			}
		})
	}
}
