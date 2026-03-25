// Package context provides helpers for bridging [scope.Scope] with
// [context.Context]. Import as a named package to avoid clashing with the
// standard context package, for example:
//
//	import scopectx "github.com/NetPo4ki/go-scope/interop/context"
package context

import (
	stdctx "context"

	"github.com/NetPo4ki/go-scope/scope"
)

// ScopeContext returns the cancellation context owned by s.
// It is equivalent to s.Context() and exists for clarity at call sites that
// work primarily with context.Context.
func ScopeContext(s *scope.Scope) stdctx.Context {
	if s == nil {
		return nil
	}
	return s.Context()
}

// LinkedContext returns a child of parent that is also canceled when link is
// done. If parent is nil, [stdctx.Background] is used. If link is nil, the
// result is [stdctx.WithCancel](parent) with no extra linkage.
func LinkedContext(parent, link stdctx.Context) (stdctx.Context, stdctx.CancelFunc) {
	if parent == nil {
		parent = stdctx.Background()
	}
	ctx, cancel := stdctx.WithCancel(parent)
	if link == nil {
		return ctx, cancel
	}
	stdctx.AfterFunc(link, cancel)
	return ctx, cancel
}
