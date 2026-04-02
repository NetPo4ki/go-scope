package otel

import (
	"context"
	"sync"
	"time"
)

// EventKind classifies recorded [Observer] callbacks for testing and debugging.
type EventKind int

const (
	EventScopeCreated EventKind = iota
	EventScopeCancelled
	EventScopeJoined
	EventTaskStarted
	EventTaskFinished
)

// Event is one lifecycle notification observed by [Recorder].
type Event struct {
	Kind     EventKind
	Cause    error
	Wait     time.Duration
	TaskDur  time.Duration
	TaskErr  error
	Panicked bool
}

// Recorder implements the scope observer contract with a thread-safe event log.
// It is intended for tests and lightweight diagnostics without OpenTelemetry SDK imports.
type Recorder struct {
	mu     sync.Mutex
	events []Event
}

// NewRecorder returns an empty [Recorder].
func NewRecorder() *Recorder { return &Recorder{} }

// Events returns a copy of recorded events.
func (r *Recorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.events))
	copy(out, r.events)
	return out
}

// ScopeCreated records scope creation.
func (r *Recorder) ScopeCreated(context.Context) {
	r.append(Event{Kind: EventScopeCreated})
}

// ScopeCancelled records scope cancellation.
func (r *Recorder) ScopeCancelled(_ context.Context, cause error) {
	r.append(Event{Kind: EventScopeCancelled, Cause: cause})
}

// ScopeJoined records scope join completion.
func (r *Recorder) ScopeJoined(_ context.Context, wait time.Duration) {
	r.append(Event{Kind: EventScopeJoined, Wait: wait})
}

// TaskStarted records task start.
func (r *Recorder) TaskStarted(context.Context) {
	r.append(Event{Kind: EventTaskStarted})
}

// TaskFinished records task completion.
func (r *Recorder) TaskFinished(_ context.Context, dur time.Duration, err error, panicked bool) {
	r.append(Event{Kind: EventTaskFinished, TaskDur: dur, TaskErr: err, Panicked: panicked})
}

func (r *Recorder) append(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}
