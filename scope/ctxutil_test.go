package scope

import (
	"context"
	"testing"
	"time"
)

func TestDeriveContextDeadlinePrecedence(t *testing.T) {
	t.Parallel()
	ctx, cancel := deriveContext(context.Background(), time.Now().Add(20*time.Millisecond), time.Second)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}
	if time.Until(deadline) > 200*time.Millisecond {
		t.Fatalf("expected explicit deadline precedence, got %v", deadline)
	}
}

func TestDeriveContextUsesTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := deriveContext(context.Background(), time.Time{}, 25*time.Millisecond)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected timeout to set deadline")
	}
	if d := time.Until(deadline); d <= 0 || d > 200*time.Millisecond {
		t.Fatalf("unexpected timeout-derived deadline: %v", d)
	}
}

func TestDeriveContextNilParent(t *testing.T) {
	t.Parallel()
	ctx, cancel := deriveContext(nil, time.Time{}, 0)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatal("expected active context before cancel")
	default:
	}
	cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected context to be canceled")
	}
}
