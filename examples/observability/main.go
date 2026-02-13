// Package main demonstrates attaching a custom Observer to a scope.
package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

type counterObserver struct {
	tasksStarted  int64
	tasksFinished int64
}

func (o *counterObserver) ScopeCreated(_ context.Context) {}
func (o *counterObserver) ScopeCancelled(_ context.Context, cause error) {
	fmt.Println("cancel cause:", cause)
}
func (o *counterObserver) ScopeJoined(_ context.Context, wait time.Duration) {
	fmt.Println("join latency:", wait)
}
func (o *counterObserver) TaskStarted(_ context.Context) { atomic.AddInt64(&o.tasksStarted, 1) }
func (o *counterObserver) TaskFinished(_ context.Context, _ time.Duration, _ error, _ bool) {
	atomic.AddInt64(&o.tasksFinished, 1)
}

func main() {
	fmt.Println("== scope + observer ==")
	withScope()
	fmt.Println("== errgroup (no observer) ==")
	withErrgroup()
	fmt.Println("== bare goroutines (no observer) ==")
	withBare()
}

func withScope() {
	obs := &counterObserver{}
	s := scope.New(context.Background(), scope.FailFast, scope.WithObserver(obs))
	s.Go(func(_ context.Context) error { time.Sleep(30 * time.Millisecond); return nil })
	s.Go(func(_ context.Context) error { time.Sleep(50 * time.Millisecond); return nil })
	_ = s.Wait()
	fmt.Printf("scope counters: started=%d finished=%d\n", obs.tasksStarted, obs.tasksFinished)
}

func withErrgroup() {
	g, _ := errgroup.WithContext(context.Background())
	var started, finished int64
	g.Go(func() error {
		atomic.AddInt64(&started, 1)
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt64(&finished, 1)
		return nil
	})
	g.Go(func() error {
		atomic.AddInt64(&started, 1)
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt64(&finished, 1)
		return nil
	})
	_ = g.Wait()
	fmt.Printf("errgroup counters: started=%d finished=%d\n", started, finished)
}

func withBare() {
	var wg sync.WaitGroup
	var started, finished int64
	wg.Add(1)
	go func() {
		defer wg.Done()
		atomic.AddInt64(&started, 1)
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt64(&finished, 1)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		atomic.AddInt64(&started, 1)
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt64(&finished, 1)
	}()
	wg.Wait()
	fmt.Printf("bare counters: started=%d finished=%d\n", started, finished)
}
