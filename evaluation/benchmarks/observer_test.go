package benchmarks

// B-4: Observer overhead — measures the cost of scope's observability hooks.
//
// Same workload (100 tasks, happy path) with three configs:
//   - no observer
//   - nop observer (interface dispatch only)
//   - counting observer (atomic increments)

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NetPo4ki/go-scope/scope"
)

type nopObs struct{}

func (nopObs) ScopeCreated(context.Context)                           {}
func (nopObs) ScopeCancelled(context.Context, error)                  {}
func (nopObs) ScopeJoined(context.Context, time.Duration)             {}
func (nopObs) TaskStarted(context.Context)                            {}
func (nopObs) TaskFinished(context.Context, time.Duration, error, bool) {}

type countingObs struct {
	started  atomic.Int64
	finished atomic.Int64
}

func (o *countingObs) ScopeCreated(context.Context)               {}
func (o *countingObs) ScopeCancelled(context.Context, error)      {}
func (o *countingObs) ScopeJoined(context.Context, time.Duration) {}
func (o *countingObs) TaskStarted(context.Context)                { o.started.Add(1) }
func (o *countingObs) TaskFinished(_ context.Context, _ time.Duration, _ error, _ bool) {
	o.finished.Add(1)
}

const observerBenchN = 100

func BenchmarkObserver_NoObserver(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := scope.New(context.Background(), scope.FailFast)
		for j := 0; j < observerBenchN; j++ {
			s.Go(func(_ context.Context) error { return nil })
		}
		_ = s.Wait()
	}
}

func BenchmarkObserver_Nop(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := scope.New(context.Background(), scope.FailFast, scope.WithObserver(nopObs{}))
		for j := 0; j < observerBenchN; j++ {
			s.Go(func(_ context.Context) error { return nil })
		}
		_ = s.Wait()
	}
}

func BenchmarkObserver_Counting(b *testing.B) {
	obs := &countingObs{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := scope.New(context.Background(), scope.FailFast, scope.WithObserver(obs))
		for j := 0; j < observerBenchN; j++ {
			s.Go(func(_ context.Context) error { return nil })
		}
		_ = s.Wait()
	}
}
