// Package main demonstrates preventing zombie goroutines via scope cancellation.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// Demonstrates preventing zombie goroutines by cancellation across three styles.
func main() {
	fmt.Println("== scope ==")
	scopeVariant()
	fmt.Println("== errgroup ==")
	errgroupVariant()
	fmt.Println("== bare goroutines ==")
	bareVariant()
}

func scopeVariant() {
	s := scope.New(context.Background(), scope.FailFast)
	s.Go(func(ctx context.Context) error {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	time.Sleep(30 * time.Millisecond)
	s.Cancel(fmt.Errorf("stop"))
	_ = s.Wait()
	fmt.Println("scope: loop terminated")
}

func errgroupVariant() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	time.Sleep(30 * time.Millisecond)
	cancel()
	_ = g.Wait()
	fmt.Println("errgroup: loop terminated")
}

func bareVariant() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:

			case <-ctx.Done():
				return
			}
		}
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	wg.Wait()
	fmt.Println("bare: loop terminated")
}
