package prom

import (
	"context"
	"sync/atomic"
	"time"
)

// Metrics is a lightweight in-memory observer that maintains counters and simple sums.
// It implements the scope.Observer interface without external dependencies.
type Metrics struct {
	// tasks
	activeTasks   atomic.Int64
	tasksStarted  atomic.Int64
	tasksFinished atomic.Int64
	tasksErrored  atomic.Int64
	tasksPanicked atomic.Int64
	taskDurSumNs  atomic.Int64

	// scopes
	scopesCreated   atomic.Int64
	scopesCancelled atomic.Int64
	joins           atomic.Int64
	joinWaitSumNs   atomic.Int64
}

// New returns a new Metrics observer.
func New() *Metrics { return &Metrics{} }

// ScopeCreated records scope creation.
func (m *Metrics) ScopeCreated(_ context.Context) {
	m.scopesCreated.Add(1)
}

// ScopeCancelled records scope cancellation.
func (m *Metrics) ScopeCancelled(_ context.Context, _ error) {
	m.scopesCancelled.Add(1)
}

// ScopeJoined records a join and accumulates wait time.
func (m *Metrics) ScopeJoined(_ context.Context, wait time.Duration) {
	m.joins.Add(1)
	m.joinWaitSumNs.Add(wait.Nanoseconds())
}

// TaskStarted increments active and started counters.
func (m *Metrics) TaskStarted(_ context.Context) {
	m.activeTasks.Add(1)
	m.tasksStarted.Add(1)
}

// TaskFinished decrements active, increments finished, and tracks error/panic and duration.
func (m *Metrics) TaskFinished(_ context.Context, dur time.Duration, err error, panicked bool) {
	m.activeTasks.Add(-1)
	m.tasksFinished.Add(1)
	if err != nil {
		m.tasksErrored.Add(1)
	}
	if panicked {
		m.tasksPanicked.Add(1)
	}
	m.taskDurSumNs.Add(dur.Nanoseconds())
}

// Snapshot exposes a copy of current metric values for exporting/inspection.
type Snapshot struct {
	ActiveTasks     int64
	TasksStarted    int64
	TasksFinished   int64
	TasksErrored    int64
	TasksPanicked   int64
	TaskDurSumNs    int64
	ScopesCreated   int64
	ScopesCancelled int64
	Joins           int64
	JoinWaitSumNs   int64
}

// GetSnapshot returns the current metrics snapshot.
func (m *Metrics) GetSnapshot() Snapshot {
	return Snapshot{
		ActiveTasks:     m.activeTasks.Load(),
		TasksStarted:    m.tasksStarted.Load(),
		TasksFinished:   m.tasksFinished.Load(),
		TasksErrored:    m.tasksErrored.Load(),
		TasksPanicked:   m.tasksPanicked.Load(),
		TaskDurSumNs:    m.taskDurSumNs.Load(),
		ScopesCreated:   m.scopesCreated.Load(),
		ScopesCancelled: m.scopesCancelled.Load(),
		Joins:           m.joins.Load(),
		JoinWaitSumNs:   m.joinWaitSumNs.Load(),
	}
}
