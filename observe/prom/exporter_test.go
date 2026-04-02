package prom

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/NetPo4ki/go-scope/scope"
)

func TestExporterCounts(t *testing.T) {
	t.Parallel()
	reg := prometheus.NewPedanticRegistry()
	exp, err := NewExporter(reg)
	if err != nil {
		t.Fatal(err)
	}

	s := scope.New(context.Background(), scope.Supervisor, scope.WithObserver(exp))
	s.Go(func(_ context.Context) error { return nil })
	s.Go(func(_ context.Context) error { return errors.New("x") })
	if err := s.Wait(); err == nil {
		t.Fatal("expected error")
	}

	if v := testutil.ToFloat64(exp.tasksStarted); v != 2 {
		t.Fatalf("tasks_started: want 2 got %v", v)
	}
	if v := testutil.ToFloat64(exp.tasksCompleted); v != 2 {
		t.Fatalf("tasks_completed: want 2 got %v", v)
	}
	if v := testutil.ToFloat64(exp.tasksFailed); v != 1 {
		t.Fatalf("tasks_failed: want 1 got %v", v)
	}
	if v := testutil.ToFloat64(exp.tasksCanceled); v != 0 {
		t.Fatalf("tasks_canceled: want 0 got %v", v)
	}
	if v := testutil.ToFloat64(exp.activeTasks); v != 0 {
		t.Fatalf("active_tasks: want 0 got %v", v)
	}
	if v := testutil.ToFloat64(exp.activeScopes); v != 0 {
		t.Fatalf("active_scopes: want 0 got %v", v)
	}

	// Histogram and counter families exist in the registry
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"scope_" + MetricTasksStartedTotal:   false,
		"scope_" + MetricTasksCompletedTotal: false,
		"scope_" + MetricTaskDurationSeconds: false,
		"scope_" + MetricJoinLatencySeconds:  false,
	}
	for _, mf := range mfs {
		n := mf.GetName()
		if _, ok := want[n]; ok {
			want[n] = true
		}
	}
	for n, seen := range want {
		if !seen {
			t.Fatalf("missing metric family %q in registry gather", n)
		}
	}
}

func TestExporterTaskCanceledCounter(t *testing.T) {
	t.Parallel()
	reg := prometheus.NewPedanticRegistry()
	exp, err := NewExporter(reg)
	if err != nil {
		t.Fatal(err)
	}

	s := scope.New(context.Background(), scope.FailFast, scope.WithObserver(exp))
	s.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	s.Cancel(context.Canceled)
	_ = s.Wait()

	if v := testutil.ToFloat64(exp.tasksCanceled); v != 1 {
		t.Fatalf("tasks_canceled: want 1 got %v", v)
	}
}

func TestExporterDoubleRegisterFails(t *testing.T) {
	t.Parallel()
	reg := prometheus.NewPedanticRegistry()
	if _, err := NewExporter(reg); err != nil {
		t.Fatal(err)
	}
	if _, err := NewExporter(reg); err == nil {
		t.Fatal("expected second NewExporter to fail with duplicate registration")
	}
}
