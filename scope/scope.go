package scope

import (
	"context"
	"fmt"
	"sync"
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

	opts Options
	obs  Observer
	lim  Limiter
}

// New creates a Scope with the given parent context, policy, and options.
func New(parent context.Context, policy Policy, optFns ...Option) *Scope {
	if parent == nil {
		parent = context.Background()
	}
	// collect options first
	base := parent
	ctx, cancel := context.WithCancel(base)
	s := &Scope{ctx: ctx, cancel: cancel, policy: policy, opts: defaultOptions()}
	for _, fn := range optFns {
		fn(&s.opts)
	}
	// apply deadline/timeout if provided
	if !s.opts.Deadline.IsZero() {
		ctx, cancel = context.WithDeadline(base, s.opts.Deadline)
		s.ctx, s.cancel = ctx, cancel
	} else if s.opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(base, s.opts.Timeout)
		s.ctx, s.cancel = ctx, cancel
	}
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

// Go starts a task owned by the Scope. The task should be cooperative and check ctx.Done().
func (s *Scope) Go(fn func(ctx context.Context) error) {
	if fn == nil {
		return
	}
	s.wg.Add(1)
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
					err := fmt.Errorf("panic: %v", r)
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

	if !wasCanceled {
		s.cancel()
		if s.obs != nil {
			s.obs.ScopeCancelled(s.ctx, cause)
		}
	} else {
		s.cancel()
	}
}

// Wait blocks until all owned tasks complete and returns the recorded error, if any.
func (s *Scope) Wait() error {
	var start time.Time
	if s.obs != nil {
		start = time.Now()
	}
	s.wg.Wait()
	if s.obs != nil {
		s.obs.ScopeJoined(s.ctx, time.Since(start))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.firstErr
}

func (s *Scope) fail(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
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
	childOpts := s.opts
	for _, fn := range optFns {
		fn(&childOpts)
	}
	ctx, cancel := context.WithCancel(s.ctx)
	cs := &Scope{ctx: ctx, cancel: cancel, policy: policy, opts: childOpts, obs: childOpts.Observer}
	if cs.obs != nil {
		cs.obs.ScopeCreated(ctx)
	}
	return cs
}
