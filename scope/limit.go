package scope

import "context"

type Limiter interface {
	Acquire(ctx context.Context) error
	Release()
}

type semLimiter struct {
	ch chan struct{}
}

func newSemaphoreLimiter(n int) Limiter {
	if n <= 0 {
		return nil
	}
	return &semLimiter{ch: make(chan struct{}, n)}
}

func (l *semLimiter) Acquire(ctx context.Context) error {
	select {
	case l.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *semLimiter) Release() {
	select {
	case <-l.ch:
	default:
	}
}
