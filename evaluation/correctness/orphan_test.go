package correctness

// CT-4: Orphan child goroutines — parent finishes before children.
//
// Demonstrates that bare Go has no concept of ownership hierarchy:
// parent WaitGroup knows nothing about children. Scope's Child()
// guarantees parent.Wait() joins all child tasks.

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NetPo4ki/go-scope/scope"
)

// TestOrphan_Bare_ParentFinishesBeforeChild shows that with bare Go,
// the "parent" WaitGroup returns while "child" goroutines are still running.
func TestOrphan_Bare_ParentFinishesBeforeChild(t *testing.T) {
	t.Parallel()
	var childDone atomic.Bool

	var parentWg sync.WaitGroup
	parentWg.Add(1)
	go func() {
		defer parentWg.Done()
		// Spawn a "child" goroutine — parent WG doesn't know about it.
		go func() {
			time.Sleep(80 * time.Millisecond)
			childDone.Store(true)
		}()
	}()

	parentWg.Wait()

	if childDone.Load() {
		t.Fatal("child should NOT be done yet — parent didn't wait for it")
	}
	t.Log("BARE: parent finished while child still running (orphan)")

	// Wait for child to finish so the test is clean.
	time.Sleep(100 * time.Millisecond)
}

// TestOrphan_Scope_ParentWaitsForChild shows that scope.Child() makes the
// parent Wait block until all child tasks complete.
func TestOrphan_Scope_ParentWaitsForChild(t *testing.T) {
	t.Parallel()
	var childDone atomic.Bool

	parent := scope.New(context.Background(), scope.FailFast)
	child := parent.Child(scope.Supervisor)
	child.Go(func(ctx context.Context) error {
		select {
		case <-time.After(80 * time.Millisecond):
			childDone.Store(true)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	if err := parent.Wait(); err != nil {
		t.Fatalf("unexpected parent error: %v", err)
	}

	if !childDone.Load() {
		t.Fatal("child should be done — parent.Wait must join child scope")
	}
	t.Log("SCOPE: parent.Wait() joined child — no orphans")
}
