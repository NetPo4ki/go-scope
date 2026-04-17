package expressiveness

// EX-1: Happy path fan-out — launch N tasks, wait for all to complete.
//
// Metrics (counted below each implementation):
//   SLOC  — source lines of code (non-blank, non-comment)
//   SYNC  — manual sync primitives used (WaitGroup, Mutex, Once, atomic)
//   CANCEL — explicit cancel/Done calls
//   BUGS  — sites where forgetting a call causes a bug

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// HappyBare runs N tasks with bare Go.
//
// SLOC=11 | SYNC=1 (WaitGroup) | CANCEL=0 | BUGS=2 (Add/Done mismatch, missing Wait)
func HappyBare(tasks []func()) {
	var wg sync.WaitGroup
	for _, fn := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}
	wg.Wait()
}

// HappyErrgroup runs N tasks with errgroup.
//
// SLOC=8 | SYNC=0 | CANCEL=0 | BUGS=1 (missing Wait)
func HappyErrgroup(ctx context.Context, tasks []func() error) error {
	g, _ := errgroup.WithContext(ctx)
	for _, fn := range tasks {
		g.Go(fn)
	}
	return g.Wait()
}

// HappyScope runs N tasks with scope.
//
// SLOC=7 | SYNC=0 | CANCEL=0 | BUGS=0 (Go after Wait is no-op, panic is caught)
func HappyScope(ctx context.Context, tasks []func(context.Context) error) error {
	s := scope.New(ctx, scope.FailFast)
	for _, fn := range tasks {
		s.Go(fn)
	}
	return s.Wait()
}
