// Package errgroup provides an adapter that mimics golang.org/x/sync/errgroup
// semantics using the local scope implementation. It enables incremental
// migration without pulling errgroup into the core library.
package errgroup

import (
	"context"

	"github.com/NetPo4ki/go-scope/scope"
)

// Group is an errgroup-like wrapper over scope.Scope (FailFast).
type Group struct {
	s   *scope.Scope
	ctx context.Context
}

// WithContext creates a Group bound to ctx. Returned context is canceled when
// any function passed to Go returns a non-nil error.
func WithContext(ctx context.Context) (*Group, context.Context) {
	s := scope.New(ctx, scope.FailFast)
	g := &Group{s: s, ctx: s.Context()}
	return g, g.ctx
}

// Go starts a function. It should return a non-nil error to signal failure.
func (g *Group) Go(f func() error) {
	if f == nil {
		return
	}
	g.s.Go(func(context.Context) error {
		return f()
	})
}

// Wait blocks until all functions have returned. It returns the first non-nil
// error (FailFast semantics) or nil on success.
func (g *Group) Wait() error {
	return g.s.Wait()
}
