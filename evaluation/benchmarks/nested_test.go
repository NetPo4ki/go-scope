package benchmarks

// B-5: Nested scopes — measures the overhead of hierarchical task ownership.
//
// Depth = 1, 2, 3 with 10 tasks per level.
// Bare Go baseline: manually nested WaitGroups + contexts.

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/NetPo4ki/go-scope/scope"
)

const tasksPerLevel = 10

func BenchmarkNested_Bare(b *testing.B) {
	for _, depth := range []int{1, 2, 3} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithCancel(context.Background())
				bareNested(ctx, depth)
				cancel()
			}
		})
	}
}

func bareNested(ctx context.Context, depth int) {
	var wg sync.WaitGroup
	for j := 0; j < tasksPerLevel; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
		}()
	}
	if depth > 1 {
		childCtx, childCancel := context.WithCancel(ctx)
		wg.Add(1)
		go func() {
			defer wg.Done()
			bareNested(childCtx, depth-1)
			childCancel()
		}()
	}
	wg.Wait()
}

func BenchmarkNested_Scope(b *testing.B) {
	for _, depth := range []int{1, 2, 3} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				root := scope.New(context.Background(), scope.FailFast)
				scopeNested(root, depth)
				_ = root.Wait()
			}
		})
	}
}

func scopeNested(parent *scope.Scope, depth int) {
	for j := 0; j < tasksPerLevel; j++ {
		parent.Go(func(_ context.Context) error { return nil })
	}
	if depth > 1 {
		child := parent.Child(scope.FailFast)
		scopeNested(child, depth-1)
	}
}
