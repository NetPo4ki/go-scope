package correctness

// CT-3: Lost errors — only the first error is captured.
//
// Demonstrates that bare Go and errgroup lose concurrent errors
// while scope's Supervisor policy aggregates all of them.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

var (
	errAlpha = errors.New("alpha")
	errBeta  = errors.New("beta")
	errGamma = errors.New("gamma")
)

// TestLostErr_Bare_OnlyFirstCaptured shows that the typical bare-Go pattern
// (mutex + first-error check) silently drops subsequent errors.
//
// Note: bare Go CAN collect all errors using mutex + []error + errors.Join
// (see expressiveness/supervisor.go for that pattern). This test demonstrates
// the common single-error pattern to show the default developer experience.
func TestLostErr_Bare_OnlyFirstCaptured(t *testing.T) {
	t.Parallel()
	var (
		mu       sync.Mutex
		firstErr error
		wg       sync.WaitGroup
	)
	errs := []error{errAlpha, errBeta, errGamma}
	for _, e := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			if firstErr == nil {
				firstErr = e
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if firstErr == nil {
		t.Fatal("expected at least one error")
	}
	captured := 0
	for _, e := range errs {
		if errors.Is(firstErr, e) {
			captured++
		}
	}
	t.Logf("BARE: captured %d of %d errors (first=%v)", captured, len(errs), firstErr)
	if captured > 1 {
		t.Fatal("bare Go should only capture one error with this pattern")
	}
}

// TestLostErr_Errgroup_OnlyFirstCaptured shows that errgroup returns only the
// first non-nil error — all others are silently dropped.
func TestLostErr_Errgroup_OnlyFirstCaptured(t *testing.T) {
	t.Parallel()
	g, _ := errgroup.WithContext(context.Background())
	errs := []error{errAlpha, errBeta, errGamma}
	for _, e := range errs {
		g.Go(func() error {
			time.Sleep(5 * time.Millisecond)
			return e
		})
	}
	result := g.Wait()
	if result == nil {
		t.Fatal("expected at least one error")
	}
	captured := 0
	for _, e := range errs {
		if errors.Is(result, e) {
			captured++
		}
	}
	t.Logf("ERRGROUP: captured %d of %d errors (result=%v)", captured, len(errs), result)
	if captured > 1 {
		t.Fatal("errgroup should only capture the first error")
	}
}

// TestLostErr_Scope_SupervisorAggregatesAll shows that scope's Supervisor
// policy collects every error via errors.Join.
func TestLostErr_Scope_SupervisorAggregatesAll(t *testing.T) {
	t.Parallel()
	s := scope.New(context.Background(), scope.Supervisor)
	errs := []error{errAlpha, errBeta, errGamma}
	for _, e := range errs {
		s.Go(func(_ context.Context) error {
			time.Sleep(5 * time.Millisecond)
			return e
		})
	}
	result := s.Wait()
	if result == nil {
		t.Fatal("expected aggregated error")
	}
	captured := 0
	for _, e := range errs {
		if errors.Is(result, e) {
			captured++
		}
	}
	t.Logf("SCOPE: captured %d of %d errors", captured, len(errs))
	if captured != len(errs) {
		t.Fatalf("scope Supervisor should capture all %d errors, got %d", len(errs), captured)
	}
}
