package correctness

// CT-2: Panic handling — unrecovered panics crash the process.
//
// Demonstrates that bare Go and errgroup let a panicking goroutine kill the
// entire process, while scope converts it to an error.
//
// Uses the subprocess pattern: the "crash" code runs in a child process
// (via exec.Command) so the test runner itself doesn't die.

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// TestPanic_Bare_CrashesProcess re-executes itself as a subprocess that spawns
// a panicking goroutine with bare Go. The subprocess is expected to crash.
func TestPanic_Bare_CrashesProcess(t *testing.T) {
	t.Parallel()
	if os.Getenv("EVAL_CRASH_BARE") == "1" {
		done := make(chan struct{})
		go func() {
			defer close(done)
			panic("boom")
		}()
		<-done
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPanic_Bare_CrashesProcess$")
	cmd.Env = append(os.Environ(), "EVAL_CRASH_BARE=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected subprocess to crash from bare-Go panic, but it exited 0")
	}
	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() != 0 {
		t.Logf("BARE: process crashed as expected (exit=%d, output=%s)",
			e.ExitCode(), truncate(string(out), 200))
		return
	}
	t.Fatalf("unexpected error type: %v", err)
}

// TestPanic_Errgroup_CrashesProcess demonstrates the same crash with errgroup.
func TestPanic_Errgroup_CrashesProcess(t *testing.T) {
	t.Parallel()
	if os.Getenv("EVAL_CRASH_ERRGROUP") == "1" {
		g, _ := errgroup.WithContext(context.Background())
		g.Go(func() error {
			panic("boom")
		})
		_ = g.Wait()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPanic_Errgroup_CrashesProcess$")
	cmd.Env = append(os.Environ(), "EVAL_CRASH_ERRGROUP=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected subprocess to crash from errgroup panic, but it exited 0")
	}
	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() != 0 {
		t.Logf("ERRGROUP: process crashed as expected (exit=%d, output=%s)",
			e.ExitCode(), truncate(string(out), 200))
		return
	}
	t.Fatalf("unexpected error type: %v", err)
}

// TestPanic_Scope_RecoveredAsError shows that scope catches the panic and
// returns it as a regular error — process stays alive.
func TestPanic_Scope_RecoveredAsError(t *testing.T) {
	t.Parallel()
	s := scope.New(context.Background(), scope.FailFast, scope.WithPanicAsError(true))
	s.Go(func(_ context.Context) error {
		panic("boom")
	})
	err := s.Wait()
	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected panic value in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "goroutine") {
		t.Fatalf("expected stack trace in error, got: %v", err)
	}
	t.Logf("SCOPE: panic recovered as error: %s", truncate(err.Error(), 120))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
