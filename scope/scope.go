package scope

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Policy int

const (
	FailFast Policy = iota
	Supervisor
)

type Option func(*Options)

type Options struct {
	PanicAsError   bool
	Observer       Observer
	MaxConcurrency int
}

func defaultOptions() Options { return Options{PanicAsError: true} }

func WithPanicAsError(v bool) Option { return func(o *Options) { o.PanicAsError = v } }

func WithObserver(obs Observer) Option { return func(o *Options) { o.Observer = obs } }

func WithMaxConcurrency(n int) Option { return func(o *Options) { o.MaxConcurrency = n } }

type Observer interface {
	ScopeCreated(ctx context.Context)
	ScopeCancelled(ctx context.Context, cause error)
	ScopeJoined(ctx context.Context, wait time.Duration)
	TaskStarted(ctx context.Context)
	TaskFinished(ctx context.Context, dur time.Duration, err error, panicked bool)
}

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

func New(parent context.Context, policy Policy, optFns ...Option) *Scope {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	s := &Scope{ctx: ctx, cancel: cancel, policy: policy, opts: defaultOptions()}
	for _, fn := range optFns {
		fn(&s.opts)
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

func (s *Scope) Context() context.Context { return s.ctx }

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
