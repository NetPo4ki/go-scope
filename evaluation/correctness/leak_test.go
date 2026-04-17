package correctness

// CT-1: Goroutine leak — Go-after-Wait and missing-Wait scenarios.
//
// Demonstrates that bare Go and errgroup silently allow goroutine leaks
// while scope rejects post-lifecycle spawns.

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// TestLeak_Bare_GoAfterWait shows that bare Go + WaitGroup does not prevent
// spawning a goroutine after Wait returns: the goroutine becomes orphaned.
func TestLeak_Bare_GoAfterWait(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
	}()
	wg.Wait()

	// "Accidental" spawn after Wait — common bug pattern.
	// Nothing prevents this; the goroutine is now orphaned.
	var leaked atomic.Bool
	cleanup := make(chan struct{})
	go func() {
		leaked.Store(true)
		<-cleanup // blocks until we release it
	}()

	time.Sleep(30 * time.Millisecond)
	if !leaked.Load() {
		t.Fatal("leaked goroutine should be running")
	}
	// The goroutine IS running but nobody owns it — that's the bug.
	t.Log("BARE: goroutine spawned after Wait is orphaned — no mechanism to prevent this")
	close(cleanup)
}

// TestLeak_Errgroup_GoAfterWait shows that errgroup.Group silently accepts
// Go calls after Wait has returned — the goroutine is untracked.
func TestLeak_Errgroup_GoAfterWait(t *testing.T) {
	t.Parallel()

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error { return nil })
	_ = g.Wait()

	// Spawn after Wait — errgroup doesn't prevent this.
	var leaked atomic.Bool
	cleanup := make(chan struct{})
	g.Go(func() error {
		leaked.Store(true)
		<-cleanup
		return nil
	})

	time.Sleep(30 * time.Millisecond)
	if !leaked.Load() {
		t.Fatal("leaked goroutine should be running")
	}
	t.Log("ERRGROUP: goroutine spawned after Wait is orphaned — no mechanism to prevent this")
	close(cleanup)
	_ = g.Wait()
}

// TestLeak_Scope_GoAfterWaitRejected shows that scope.Go after Wait is a
// no-op: TryGo returns false and no goroutine is spawned.
func TestLeak_Scope_GoAfterWaitRejected(t *testing.T) {
	t.Parallel()

	s := scope.New(context.Background(), scope.FailFast)
	s.Go(func(_ context.Context) error { return nil })
	_ = s.Wait()

	ok := s.TryGo(func(_ context.Context) error {
		select {} // would block forever
	})
	if ok {
		t.Fatal("scope should reject Go after Wait")
	}
	t.Log("SCOPE: TryGo correctly rejected — no goroutine spawned, no leak possible")
}
