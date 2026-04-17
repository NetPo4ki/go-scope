package expressiveness

// EX-2: FailFast fan-out — first error cancels remaining tasks.
//
// Metrics counted below each implementation.

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// FailFastBare cancels siblings on first error using bare Go.
//
// SLOC=19 | SYNC=3 (WaitGroup, Once, cancel) | CANCEL=2 (cancel(), defer cancel())
// BUGS=4 (Add/Done mismatch, missing cancel, missing Once, missing Wait)
func FailFastBare(parent context.Context, tasks []func(context.Context) error) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)
	for _, fn := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(ctx); err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// FailFastErrgroup cancels siblings on first error using errgroup.
//
// SLOC=8 | SYNC=0 | CANCEL=0 | BUGS=1 (missing Wait leaks goroutines)
func FailFastErrgroup(ctx context.Context, tasks []func(context.Context) error) error {
	g, gctx := errgroup.WithContext(ctx)
	for _, fn := range tasks {
		g.Go(func() error {
			return fn(gctx)
		})
	}
	return g.Wait()
}

// FailFastScope cancels siblings on first error using scope.
//
// SLOC=7 | SYNC=0 | CANCEL=0 | BUGS=0
func FailFastScope(ctx context.Context, tasks []func(context.Context) error) error {
	s := scope.New(ctx, scope.FailFast)
	for _, fn := range tasks {
		s.Go(fn)
	}
	return s.Wait()
}
