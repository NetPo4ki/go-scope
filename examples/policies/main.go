// Package main demonstrates FailFast vs Supervisor error policies.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

func main() {
	fmt.Println("== scope.FailFast ==")
	scopeFailFast()
	fmt.Println("== scope.Supervisor ==")
	scopeSupervisor()
	fmt.Println("== errgroup (fail-fast) ==")
	egroup()
	fmt.Println("== bare goroutines (fail-fast-like) ==")
	bare()
}

func scopeFailFast() {
	ff := scope.New(context.Background(), scope.FailFast)
	ff.Go(func(_ context.Context) error { time.Sleep(20 * time.Millisecond); return errors.New("boom") })
	ff.Go(func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	fmt.Println("result:", ff.Wait())
}

func scopeSupervisor() {
	sup := scope.New(context.Background(), scope.Supervisor)
	sup.Go(func(_ context.Context) error { time.Sleep(20 * time.Millisecond); return errors.New("boom") })
	sup.Go(func(_ context.Context) error { time.Sleep(10 * time.Millisecond); return nil })
	fmt.Println("result:", sup.Wait())
}

func egroup() {
	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { time.Sleep(20 * time.Millisecond); return errors.New("boom") })
	g.Go(func() error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	fmt.Println("result:", g.Wait())
}

func bare() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error
	record := func(err error) {
		if err == nil {
			return
		}
		once.Do(func() { firstErr = err; cancel() })
	}
	wg.Add(1)
	go func() { defer wg.Done(); time.Sleep(20 * time.Millisecond); record(errors.New("boom")) }()
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-time.After(200 * time.Millisecond):
			return
		case <-ctx.Done():
			return
		}
	}()
	wg.Wait()
	fmt.Println("result:", firstErr)
}
