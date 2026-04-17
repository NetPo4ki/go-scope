package correctness

// CT-5: Hierarchical cancel propagation.
//
// Demonstrates that bare Go requires manual context wiring for multi-level
// cancellation, while scope propagates cancel from parent to child to
// grandchild automatically.

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NetPo4ki/go-scope/scope"
)

// TestCancel_Bare_ManualWiring shows the amount of plumbing needed to
// propagate cancellation through 3 levels with bare Go.
func TestCancel_Bare_ManualWiring(t *testing.T) {
	t.Parallel()
	var grandchildCanceled atomic.Bool

	rootCtx, rootCancel := context.WithCancel(context.Background())
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
				grandchildCanceled.Store(true)
			}()

			<-grandCtx.Done()
			grandWg.Wait()
		}()

		<-childCtx.Done()
		childWg.Wait()
	}()

	time.Sleep(20 * time.Millisecond)
	rootCancel()
	rootWg.Wait()

	if !grandchildCanceled.Load() {
		t.Fatal("grandchild should have been canceled")
	}
	// The test passes, but look at the code above:
	// 3 contexts, 3 cancel funcs, 3 WaitGroups, 3 defer statements,
	// nested goroutines — extremely error-prone.
	t.Log("BARE: works, but requires 3 contexts + 3 WaitGroups + manual wiring")
}

// TestCancel_Scope_AutomaticPropagation shows the same 3-level hierarchy
// with scope: cancel root → child → grandchild automatically.
func TestCancel_Scope_AutomaticPropagation(t *testing.T) {
	t.Parallel()
	var grandchildCanceled atomic.Bool

	root := scope.New(context.Background(), scope.FailFast)
	child := root.Child(scope.FailFast)
	grand := child.Child(scope.FailFast)

	grand.Go(func(ctx context.Context) error {
		<-ctx.Done()
		grandchildCanceled.Store(true)
		return ctx.Err()
	})

	time.Sleep(20 * time.Millisecond)
	root.Cancel(nil)
	_ = root.Wait()

	if !grandchildCanceled.Load() {
		t.Fatal("grandchild should have been canceled by root")
	}
	// 3 lines to create hierarchy, 1 line to cancel root — done.
	t.Log("SCOPE: root.Cancel propagated to grandchild automatically")
}
