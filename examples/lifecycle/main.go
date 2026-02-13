// Package main demonstrates scope-bound lifecycles with child scopes.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

// Demonstrates scope-bound lifecycles with a child scope and compares to errgroup/bare.
func main() {
	fmt.Println("== scope: parent/child ==")
	withScope()

	fmt.Println("== errgroup: parent cancels ==")
	withErrgroup()

	fmt.Println("== bare goroutines: manual cancel ==")
	withBare()
}

func withScope() {
	parent := scope.New(context.Background(), scope.Supervisor)
	child := parent.Child(scope.Supervisor)

	child.Go(func(ctx context.Context) error {
		select {
		case <-time.After(150 * time.Millisecond):
			fmt.Println("scope child: completed")
			return nil
		case <-ctx.Done():
			fmt.Println("scope child: canceled")
			return ctx.Err()
		}
	})

	time.AfterFunc(50*time.Millisecond, func() { parent.Cancel(fmt.Errorf("shutdown")) })
	_ = child.Wait()
	_ = parent.Wait()
}

func withErrgroup() {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		select {
		case <-time.After(150 * time.Millisecond):
			fmt.Println("errgroup child: completed")
			return nil
		case <-ctx.Done():
			fmt.Println("errgroup child: canceled")
			return ctx.Err()
		}
	})

	pctx, cancel := context.WithCancel(ctx)
	g2, cctx := errgroup.WithContext(pctx)
	g2.Go(func() error {
		select {
		case <-time.After(150 * time.Millisecond):
			fmt.Println("errgroup child2: completed")
			return nil
		case <-cctx.Done():
			fmt.Println("errgroup child2: canceled")
			return cctx.Err()
		}
	})
	time.AfterFunc(50*time.Millisecond, cancel)
	_ = g.Wait()
	_ = g2.Wait()
}

func withBare() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-time.After(150 * time.Millisecond):
			fmt.Println("bare child: completed")
		case <-ctx.Done():
			fmt.Println("bare child: canceled")
		}
	}()
	time.AfterFunc(50*time.Millisecond, cancel)
	wg.Wait()
}
