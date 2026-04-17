package expressiveness

// EX-4: Nested hierarchy — parent → child → grandchild, cancel from root.
//
// Metrics counted below each implementation.

import (
	"context"
	"sync"

	"github.com/NetPo4ki/go-scope/scope"
)

// NestedBare creates a 3-level hierarchy with bare Go and returns a function
// to cancel the root and wait for all goroutines to drain.
//
// SLOC=30 | SYNC=3 (3× WaitGroup) | CANCEL=3 (3× cancel + 3× defer)
// BUGS=6 (3× Add/Done mismatch, 3× missing cancel, orphan if any defer missed)
func NestedBare(parent context.Context) (cancel func()) {
	rootCtx, rootCancel := context.WithCancel(parent)

	var rootWg sync.WaitGroup
	rootWg.Add(1)
	go func() {
		defer rootWg.Done()
		childCtx, childCancel := context.WithCancel(rootCtx)
		defer childCancel()

		var childWg sync.WaitGroup
		childWg.Add(1)
		go func() {
			defer childWg.Done()
			grandCtx, grandCancel := context.WithCancel(childCtx)
			defer grandCancel()

			var grandWg sync.WaitGroup
			grandWg.Add(1)
			go func() {
				defer grandWg.Done()
				<-grandCtx.Done()
			}()
			grandWg.Wait()
		}()
		childWg.Wait()
	}()

	return func() {
		rootCancel()
		rootWg.Wait()
	}
}

// NestedScope creates a 3-level hierarchy with scope.
// The caller cancels via root.Cancel() and waits via root.Wait().
//
// SLOC=10 | SYNC=0 | CANCEL=0 | BUGS=0
func NestedScope(parent context.Context) *scope.Scope {
	root := scope.New(parent, scope.FailFast)
	child := root.Child(scope.FailFast)
	grand := child.Child(scope.FailFast)

	grand.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	return root
}
