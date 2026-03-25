package context

import (
	stdctx "context"
	"testing"
	"time"

	"github.com/NetPo4ki/go-scope/scope"
)

func TestScopeContextNil(t *testing.T) {
	t.Parallel()
	if ScopeContext(nil) != nil {
		t.Fatal("expected nil for nil scope")
	}
}

func TestScopeContextMatchesScope(t *testing.T) {
	t.Parallel()
	s := scope.New(stdctx.Background(), scope.FailFast)
	if got, want := ScopeContext(s), s.Context(); got != want {
		t.Fatalf("ScopeContext mismatch")
	}
}

func TestLinkedContextNilLink(t *testing.T) {
	t.Parallel()
	parent, stop := stdctx.WithCancel(stdctx.Background())
	defer stop()
	ctx, cancel := LinkedContext(parent, nil)
	defer cancel()
	if ctx == nil {
		t.Fatal("expected non-nil ctx")
	}
	stop()
	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected ctx canceled when parent canceled")
	}
}

func TestLinkedContextPropagatesLinkCancel(t *testing.T) {
	t.Parallel()
	parent := stdctx.Background()
	link, stopLink := stdctx.WithCancel(stdctx.Background())
	ctx, cancel := LinkedContext(parent, link)
	defer cancel()

	stopLink()
	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected ctx canceled when link canceled")
	}
}
