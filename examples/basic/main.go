// Package main demonstrates basic scope usage with FailFast policy.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

func main() {
	fmt.Println("== scope.FailFast ==")
	withScope()

	fmt.Println("== errgroup (fail-fast) ==")
	withErrgroup()

	fmt.Println("== bare goroutines (fail-fast) ==")
	withBare()
}

func withScope() {
	s := scope.New(context.Background(), scope.FailFast)
	s.Go(func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			fmt.Println("scope: task1 done")
			return nil
		case <-ctx.Done():
			fmt.Println("scope: task1 canceled")
			return ctx.Err()
		}
	})
	s.Go(func(_ context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return fmt.Errorf("upstream error")
	})
	fmt.Println("scope result:", s.Wait())

}

func withErrgroup() {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		select {
		case <-time.After(200 * time.Millisecond):
			fmt.Println("errgroup: task1 done")
			return nil
		case <-ctx.Done():
			fmt.Println("errgroup: task1 canceled")
			return ctx.Err()
		}
	})
	g.Go(func() error {
		time.Sleep(50 * time.Millisecond)
		return fmt.Errorf("upstream error")
	})
	fmt.Println("errgroup result:", g.Wait())
}

func withBare() {
	parent := context.Background()
	ctx, cancel := context.WithCancel(parent)
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
	go func() {
		defer wg.Done()
		select {
		case <-time.After(200 * time.Millisecond):
			fmt.Println("bare: task1 done")
		case <-ctx.Done():
			fmt.Println("bare: task1 canceled")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		record(fmt.Errorf("upstream error"))
	}()

	wg.Wait()
	fmt.Println("bare result:", firstErr)
}
