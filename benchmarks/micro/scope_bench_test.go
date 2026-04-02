package micro

import (
	"context"
	"fmt"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/scope"
)

func BenchmarkScope_SpawnWait(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("tasks_%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s := scope.New(context.Background(), scope.FailFast)
				for j := 0; j < n; j++ {
					s.Go(func(context.Context) error { return nil })
				}
				if err := s.Wait(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkErrgroup_SpawnWait(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("tasks_%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				g, _ := errgroup.WithContext(context.Background())
				for j := 0; j < n; j++ {
					g.Go(func() error { return nil })
				}
				if err := g.Wait(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkScope_FailFastFirstError(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := scope.New(context.Background(), scope.FailFast)
		s.Go(func(context.Context) error { return fmt.Errorf("e") })
		s.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		_ = s.Wait()
	}
}

func BenchmarkErrgroup_FirstError(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g, ctx := errgroup.WithContext(context.Background())
		g.Go(func() error { return fmt.Errorf("e") })
		g.Go(func() error {
			<-ctx.Done()
			return ctx.Err()
		})
		_ = g.Wait()
	}
}
