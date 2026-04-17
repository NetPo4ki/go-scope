package main

// B-2: Drain latency harness.
//
// Measures how quickly each approach shuts down N cooperative tasks after an
// error is injected. The key measurement:
//
//   T_drain = time from error injection moment until Wait() returns.
//
// Four approaches compared:
//   bare_nocancel — no fail-fast: workers run to completion despite the error
//   bare_cancel   — manual fail-fast: cancel() + sync.Once plumbing
//   errgroup      — errgroup.WithContext: built-in first-error cancel
//   scope         — scope.FailFast: automatic sibling cancellation
//
// The interesting contrast is bare_nocancel vs the rest: it shows the cost
// of NOT having fail-fast (workers waste full work duration).
//
// Usage:
//   go run ./evaluation/benchmarks/drain/ [flags]
//
// Flags:
//   -n          number of worker tasks (default 100)
//   -work       simulated work duration per task (default 5ms)
//   -samples    number of repetitions (default 200)
//   -gomaxprocs override GOMAXPROCS (default: runtime.NumCPU)
//   -out        CSV output file (default: stdout)

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

func main() {
	n := flag.Int("n", 100, "number of worker tasks")
	work := flag.Duration("work", 5*time.Millisecond, "simulated work per task")
	samples := flag.Int("samples", 200, "repetitions per approach")
	procs := flag.Int("gomaxprocs", runtime.NumCPU(), "GOMAXPROCS")
	outFile := flag.String("out", "", "CSV output file (default: stdout)")
	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	out := os.Stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot create %s: %v\n", *outFile, err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	w := csv.NewWriter(out)
	defer w.Flush()
	_ = w.Write([]string{"approach", "sample", "drain_ns"})

	fmt.Fprintf(os.Stderr, "drain-latency: N=%d work=%v samples=%d GOMAXPROCS=%d\n",
		*n, *work, *samples, *procs)

	approaches := []struct {
		name string
		fn   func(n int, work time.Duration) time.Duration
	}{
		{"bare_nocancel", func(nn int, wk time.Duration) time.Duration { return drainBareNoCancel(nn, wk) }},
		{"bare_cancel", func(nn int, wk time.Duration) time.Duration { return drainBareCancel(nn, wk) }},
		{"errgroup", func(nn int, wk time.Duration) time.Duration { return drainErrgroup(nn, wk) }},
		{"scope", func(nn int, wk time.Duration) time.Duration { return drainScope(nn, wk) }},
	}

	for _, a := range approaches {
		durations := make([]time.Duration, *samples)
		for i := range durations {
			durations[i] = a.fn(*n, *work)
			_ = w.Write([]string{a.name, strconv.Itoa(i), strconv.FormatInt(durations[i].Nanoseconds(), 10)})
		}
		printPercentiles(a.name, durations)
	}
}

// cooperativeWork simulates I/O-bound work with cooperative cancellation.
func cooperativeWork(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// nonCooperativeWork ignores context — simulates a task that cannot be canceled.
func nonCooperativeWork(d time.Duration) {
	time.Sleep(d)
}

// drainBareNoCancel: no fail-fast. Error occurs but nobody cancels workers.
// Workers run to full completion. This is the "what happens without
// structured concurrency" baseline.
func drainBareNoCancel(n int, work time.Duration) time.Duration {
	var wg sync.WaitGroup
	allStarted := make(chan struct{})

	for j := 0; j < n; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-allStarted
			nonCooperativeWork(work)
		}()
	}

	// Signal all workers to begin, then inject error and start timer.
	close(allStarted)
	// Error happens here — but nobody acts on it.
	t0 := time.Now()
	wg.Wait()
	return time.Since(t0)
}

// drainBareCancel: manual fail-fast with context + sync.Once + error recording.
// Mirrors what errgroup/scope do internally: record error, then cancel.
func drainBareCancel(n int, work time.Duration) time.Duration {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)
	allStarted := make(chan struct{})
	var injected atomic.Int64

	for j := 0; j < n; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-allStarted
			if err := cooperativeWork(ctx, work); err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-allStarted
		injected.Store(time.Now().UnixNano())
		once.Do(func() {
			firstErr = fmt.Errorf("injected")
			cancel()
		})
	}()

	close(allStarted)
	wg.Wait()
	_ = firstErr
	t0 := time.Unix(0, injected.Load())
	return time.Since(t0)
}

// drainErrgroup: errgroup.WithContext — first error cancels context.
func drainErrgroup(n int, work time.Duration) time.Duration {
	g, ctx := errgroup.WithContext(context.Background())
	allStarted := make(chan struct{})
	var injected atomic.Int64

	for j := 0; j < n; j++ {
		g.Go(func() error {
			<-allStarted
			return cooperativeWork(ctx, work)
		})
	}

	g.Go(func() error {
		<-allStarted
		injected.Store(time.Now().UnixNano())
		return fmt.Errorf("injected")
	})

	close(allStarted)
	_ = g.Wait()
	t0 := time.Unix(0, injected.Load())
	return time.Since(t0)
}

// drainScope: scope.FailFast — first error cancels all siblings.
func drainScope(n int, work time.Duration) time.Duration {
	s := scope.New(context.Background(), scope.FailFast)
	allStarted := make(chan struct{})
	var injected atomic.Int64

	for j := 0; j < n; j++ {
		s.Go(func(ctx context.Context) error {
			<-allStarted
			return cooperativeWork(ctx, work)
		})
	}

	s.Go(func(_ context.Context) error {
		<-allStarted
		injected.Store(time.Now().UnixNano())
		return fmt.Errorf("injected")
	})

	close(allStarted)
	_ = s.Wait()
	t0 := time.Unix(0, injected.Load())
	return time.Since(t0)
}

func printPercentiles(name string, durations []time.Duration) {
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	pcts := []float64{50, 90, 95, 99}
	fmt.Fprintf(os.Stderr, "\n%s (n=%d):\n", name, len(durations))
	for _, p := range pcts {
		idx := int(math.Ceil(p/100*float64(len(durations)))) - 1
		if idx < 0 {
			idx = 0
		}
		fmt.Fprintf(os.Stderr, "  p%-4.0f %v\n", p, durations[idx])
	}
	fmt.Fprintf(os.Stderr, "  mean  %v\n", mean(durations))
}

func mean(ds []time.Duration) time.Duration {
	var sum int64
	for _, d := range ds {
		sum += d.Nanoseconds()
	}
	return time.Duration(sum / int64(len(ds)))
}
