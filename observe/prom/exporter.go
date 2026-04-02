package prom

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metric name fragments used with Namespace "scope" (full names: scope_*).
const (
	MetricTasksStartedTotal   = "tasks_started_total"
	MetricTasksCompletedTotal = "tasks_completed_total"
	MetricTasksFailedTotal    = "tasks_failed_total"
	MetricTasksCanceledTotal  = "tasks_canceled_total"
	MetricActiveTasks         = "active_tasks"
	MetricActiveScopes        = "active_scopes"
	MetricTaskDurationSeconds = "task_duration_seconds"
	MetricJoinLatencySeconds  = "join_latency_seconds"
)

// Exporter implements [scope.Observer] and records metrics to a Prometheus
// [prometheus.Registerer].
type Exporter struct {
	tasksStarted   prometheus.Counter
	tasksCompleted prometheus.Counter
	tasksFailed    prometheus.Counter
	tasksCanceled  prometheus.Counter
	activeTasks    prometheus.Gauge
	activeScopes   prometheus.Gauge
	taskDuration   prometheus.Histogram
	joinLatency    prometheus.Histogram
}

// NewExporter builds an Exporter, registers its collectors with reg, and returns
// it. reg is usually a [*prometheus.Registry] from [prometheus.NewRegistry].
func NewExporter(reg prometheus.Registerer) (*Exporter, error) {
	e := &Exporter{
		tasksStarted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "scope",
			Name:      MetricTasksStartedTotal,
			Help:      "Total tasks started under scope supervision.",
		}),
		tasksCompleted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "scope",
			Name:      MetricTasksCompletedTotal,
			Help:      "Total tasks that finished (success or error).",
		}),
		tasksFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "scope",
			Name:      MetricTasksFailedTotal,
			Help:      "Total tasks that finished with a non-nil error or panic-as-error.",
		}),
		tasksCanceled: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "scope",
			Name:      MetricTasksCanceledTotal,
			Help:      "Total tasks that finished with context.Canceled.",
		}),
		activeTasks: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "scope",
			Name:      MetricActiveTasks,
			Help:      "Tasks currently running inside scopes.",
		}),
		activeScopes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "scope",
			Name:      MetricActiveScopes,
			Help:      "Scopes created but not yet joined (Wait completed).",
		}),
		taskDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "scope",
			Name:      MetricTaskDurationSeconds,
			Help:      "Task wall duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}),
		joinLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "scope",
			Name:      MetricJoinLatencySeconds,
			Help:      "Scope Wait blocking duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}),
	}

	for _, c := range []prometheus.Collector{
		e.tasksStarted, e.tasksCompleted, e.tasksFailed, e.tasksCanceled,
		e.activeTasks, e.activeScopes, e.taskDuration, e.joinLatency,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}

	return e, nil
}

// ScopeCreated implements [scope.Observer].
func (e *Exporter) ScopeCreated(context.Context) {
	e.activeScopes.Inc()
}

// ScopeCancelled implements [scope.Observer].
func (e *Exporter) ScopeCancelled(context.Context, error) {
	// Scope-level cancel does not increment task_* counters; tasks record outcome.
}

// ScopeJoined implements [scope.Observer].
func (e *Exporter) ScopeJoined(_ context.Context, wait time.Duration) {
	e.joinLatency.Observe(wait.Seconds())
	e.activeScopes.Dec()
}

// TaskStarted implements [scope.Observer].
func (e *Exporter) TaskStarted(context.Context) {
	e.activeTasks.Inc()
	e.tasksStarted.Inc()
}

// TaskFinished implements [scope.Observer].
func (e *Exporter) TaskFinished(_ context.Context, dur time.Duration, err error, panicked bool) {
	e.activeTasks.Dec()
	e.tasksCompleted.Inc()
	e.taskDuration.Observe(dur.Seconds())
	if err != nil || panicked {
		e.tasksFailed.Inc()
	}
	if err != nil && errors.Is(err, context.Canceled) {
		e.tasksCanceled.Inc()
	}
}
