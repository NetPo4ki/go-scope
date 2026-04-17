package scope

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Policy controls error propagation behavior in a Scope.
type Policy int

const (
	// FailFast cancels siblings on the first task error or panic and records the cause.
	FailFast Policy = iota
	// Supervisor allows siblings to continue despite a task error; errors may be aggregated.
	Supervisor
)

// Option configures a Scope at construction time.
type Option func(*Options)

// Options holds optional settings for Scope construction.
type Options struct {
	// PanicAsError converts a panic inside a task to an error when true.
	PanicAsError bool
	// Observer receives lifecycle events; if nil, hooks are skipped (near-zero overhead).
	Observer Observer
	// MaxConcurrency bounds concurrent tasks in a scope when > 0.
	MaxConcurrency int
	// Timeout applies a relative deadline to the scope when > 0 (ignored if Deadline is set).
	Timeout time.Duration
	// Deadline applies an absolute deadline to the scope when non-zero.
	Deadline time.Time
}

func defaultOptions() Options { return Options{PanicAsError: true} }

// WithPanicAsError toggles converting task panics into errors.
func WithPanicAsError(v bool) Option { return func(o *Options) { o.PanicAsError = v } }

// WithObserver attaches an observer for metrics/tracing hooks (nil = disabled).
func WithObserver(obs Observer) Option { return func(o *Options) { o.Observer = obs } }

// WithMaxConcurrency limits the number of concurrent tasks in a scope (n>0).
func WithMaxConcurrency(n int) Option { return func(o *Options) { o.MaxConcurrency = n } }

// WithTimeout applies a relative deadline to the scope (ignored if WithDeadline is also set).
func WithTimeout(d time.Duration) Option { return func(o *Options) { o.Timeout = d } }

// WithDeadline applies an absolute deadline to the scope.
func WithDeadline(t time.Time) Option { return func(o *Options) { o.Deadline = t } }

// Observer receives lifecycle events for metrics/tracing.
type Observer interface {
	ScopeCreated(ctx context.Context)
	ScopeCancelled(ctx context.Context, cause error)
	ScopeJoined(ctx context.Context, wait time.Duration)
	TaskStarted(ctx context.Context)
	TaskFinished(ctx context.Context, dur time.Duration, err error, panicked bool)
}

// Scope owns a set of tasks and provides an explicit join point via Wait.
type Scope struct {
	ctx      context.Context
	cancel   context.CancelFunc
	policy   Policy
	wg       sync.WaitGroup
	mu       sync.Mutex
	firstErr error
	canceled bool
	waiting  bool
	done     bool

	// cancelDone is set atomically after Cancel() has recorded the error
	// and called s.cancel(). Used as a lock-free fast path in fail() to
	// avoid mutex contention when many goroutines drain simultaneously.
	cancelDone atomic.Uint32

	opts Options
	obs  Observer
	lim  Limiter
	errs []error
}

// New creates a Scope with the given parent context, policy, and options.
func New(parent context.Context, policy Policy, optFns ...Option) *Scope {
	if parent == nil {
		parent = context.Background()
	}
	// collect options first
	s := &Scope{policy: policy, opts: defaultOptions()}
	for _, fn := range optFns {
		fn(&s.opts)
	}

	ctx, cancel := deriveContext(parent, s.opts.Deadline, s.opts.Timeout)
	s.ctx, s.cancel = ctx, cancel
	s.obs = s.opts.Observer
	if s.opts.MaxConcurrency > 0 {
		s.lim = newSemaphoreLimiter(s.opts.MaxConcurrency)
	}
	if s.obs != nil {
		s.obs.ScopeCreated(ctx)
	}
	return s
}

// Context returns the Scope's context.
func (s *Scope) Context() context.Context { return s.ctx }

// Go starts a task owned by the Scope.
//
// Go is best-effort: if the scope is already canceled, waiting, or done, the
// task is not started and the call is a no-op. Use TryGo when the caller needs
// to know whether spawning succeeded.
func (s *Scope) Go(fn func(ctx context.Context) error) {
	_ = s.TryGo(fn)
}

// TryGo starts a task owned by the Scope and reports whether spawning
// succeeded.
//
// TryGo returns false when fn is nil or when the scope is no longer accepting
// new tasks (already canceled, waiting, or done).
func (s *Scope) TryGo(fn func(ctx context.Context) error) bool {
	if fn == nil {
		return false
	}
	s.mu.Lock()
	if s.waiting || s.done || s.canceled {
		s.mu.Unlock()
		return false
	}
	s.wg.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wg.Done()
		if s.lim != nil {
			if err := s.lim.Acquire(s.ctx); err != nil {
				s.fail(err)
				return
			}
			defer s.lim.Release()
		}
		defer func() {
			if r := recover(); r != nil {
				if s.opts.PanicAsError {
					err := panicToError(r)
					s.fail(err)
					if s.obs != nil {
						s.obs.TaskFinished(s.ctx, 0, err, true)
					}
				} else {
					if s.obs != nil {
						s.obs.TaskFinished(s.ctx, 0, nil, true)
					}
					panic(r)
				}
			}
		}()

		var start time.Time
		if s.obs != nil {
			start = time.Now()
			s.obs.TaskStarted(s.ctx)
		}

		err := fn(s.ctx)
		if err != nil {
			s.fail(err)
		}
		if s.obs != nil {
			s.obs.TaskFinished(s.ctx, time.Since(start), err, false)
		}
	}()
	return true
}

// Cancel cancels the Scope and records the first non-nil error as the cause.
func (s *Scope) Cancel(err error) {
	s.mu.Lock()
	wasCanceled := s.canceled
	s.canceled = true
	if s.firstErr == nil && err != nil {
		s.firstErr = err
	}
	cause := s.firstErr
	s.mu.Unlock()

	s.cancel()
	s.cancelDone.Store(1)

	if !wasCanceled && s.obs != nil {
		s.obs.ScopeCancelled(s.ctx, cause)
	}
}

// Wait blocks until all owned tasks complete and returns the recorded error, if any.
func (s *Scope) Wait() error {
	var start time.Time
	if s.obs != nil {
		start = time.Now()
	}
	s.mu.Lock()
	s.waiting = true
	s.mu.Unlock()
	s.wg.Wait()
	s.mu.Lock()
	s.done = true
	s.mu.Unlock()
	if s.obs != nil {
		s.obs.ScopeJoined(s.ctx, time.Since(start))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.policy == Supervisor && len(s.errs) > 0 {
		return errors.Join(s.errs...)
	}
	return s.firstErr
}

func (s *Scope) fail(err error) {
	if err == nil {
		return
	}
	// Lock-free fast path for FailFast: once Cancel() has completed
	// (firstErr recorded, context canceled), subsequent fail() calls
	// have no observable effect — skip the mutex entirely.
	if s.policy == FailFast && s.cancelDone.Load() == 1 {
		return
	}
	s.mu.Lock()
	if s.policy == Supervisor {
		s.errs = append(s.errs, err)
	}
	if s.firstErr == nil {
		s.firstErr = err
	}
	shouldCancel := s.policy == FailFast
	cause := s.firstErr
	s.mu.Unlock()
	if shouldCancel {
		s.Cancel(cause)
	}
}

// Child creates a child Scope inheriting options; parent cancellation cancels the child.
func (s *Scope) Child(policy Policy, optFns ...Option) *Scope {
	s.mu.Lock()
	if s.waiting || s.done {
		s.mu.Unlock()
		ctx, cancel := context.WithCancel(s.ctx)
		cancel()
		return &Scope{
			ctx:    ctx,
			cancel: cancel,
			policy: policy,
			opts:   defaultOptions(),
		}
	}
	s.wg.Add(1)
	s.mu.Unlock()

	childOpts := s.opts
	for _, fn := range optFns {
		fn(&childOpts)
	}
	ctx, cancel := deriveContext(s.ctx, childOpts.Deadline, childOpts.Timeout)
	cs := &Scope{ctx: ctx, cancel: cancel, policy: policy, opts: childOpts, obs: childOpts.Observer}
	if childOpts.MaxConcurrency > 0 {
		cs.lim = newSemaphoreLimiter(childOpts.MaxConcurrency)
	}
	if cs.obs != nil {
		cs.obs.ScopeCreated(ctx)
	}

	go func() {
		defer s.wg.Done()
		if err := cs.Wait(); err != nil {
			s.fail(err)
		}
	}()

	return cs
}
