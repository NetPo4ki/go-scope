package expressiveness

// EX-3: Supervisor — errors don't cancel siblings, collect all errors.
//
// Metrics counted below each implementation.

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// SupervisorBare collects all errors using bare Go (mutex + slice).
//
// SLOC=17 | SYNC=2 (WaitGroup, Mutex) | CANCEL=0
// BUGS=3 (Add/Done mismatch, missing Lock, missing Wait)
func SupervisorBare(tasks []func() error) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, fn := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// SupervisorErrgroup — errgroup CANNOT do supervisor semantics.
// It cancels the context on first error and only returns the first error.
// This implementation is the closest approximation: ignore the context cancel.
//
// SLOC=8 | SYNC=0 | CANCEL=0 | BUGS=N/A (fundamentally cannot aggregate errors)
func SupervisorErrgroup(ctx context.Context, tasks []func() error) error {
	g, _ := errgroup.WithContext(ctx)
	for _, fn := range tasks {
		g.Go(fn)
	}
	return g.Wait() // only first error returned
}

// SupervisorScope collects all errors using scope's Supervisor policy.
//
// SLOC=7 | SYNC=0 | CANCEL=0 | BUGS=0
func SupervisorScope(ctx context.Context, tasks []func(context.Context) error) error {
	s := scope.New(ctx, scope.Supervisor)
	for _, fn := range tasks {
		s.Go(fn)
	}
	return s.Wait() // errors.Join of all errors
}
