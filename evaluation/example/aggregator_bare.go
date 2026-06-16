package example

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/NetPo4ki/go-scope/evaluation/example/backends"
)

// BareGoAggregator implements the dashboard handler using only the Go
// standard library: context.WithTimeout for the SLA, a buffered channel as
// a concurrency-limiting semaphore, sync.WaitGroup to join, sync.Mutex to
// guard partial results, and explicit deferred recover() in every
// goroutine for panic safety.
//
// This is the implementation a competent Go developer writes today when
// they are aware of the failure modes. It is also the implementation that
// the Go bug studies (Tu et al., Chabbi et al.) catalogue as the dominant
// source of leaked-goroutine and lost-error defects, because every line
// involving sync, channels, recover, or cancellation is a place the
// developer can get it wrong.
//
// The implementation is offered as the comparison baseline, not as an
// example of how to write Go.
type BareGoAggregator struct {
	B *backends.Backends
}

func (a *BareGoAggregator) Aggregate(ctx context.Context, cfg Config) (Dashboard, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.SLA)
	defer cancel()

	// Concurrency limiter: a buffered channel used as a counting semaphore.
	// Acquire blocks until a slot is free or ctx is cancelled. Release
	// must be unconditional or the slot leaks.
	sem := make(chan struct{}, cfg.MaxConc)
	acquire := func(ctx context.Context) error {
		select {
		case sem <- struct{}{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	release := func() { <-sem }

	var (
		wg sync.WaitGroup
		mu sync.Mutex // protects the result fields below

		profile      backends.Profile
		posts        []backends.Post
		subscription backends.Subscription
		recs         []backends.Recommendation
		notifCount   int = -1 // -1 = optional fetch failed
	)

	// requiredErr is the first required-field error observed. Once set,
	// cancel() is called so siblings stop. We use sync.Once-style
	// idempotence via mu+nil-check rather than sync.Once because we also
	// need to capture the value, not just a flag.
	var requiredErr error
	failRequired := func(err error) {
		mu.Lock()
		if requiredErr == nil {
			requiredErr = err
			cancel()
		}
		mu.Unlock()
	}

	// runRequired wraps a required-field task with concurrency limiting,
	// panic recovery, and fail-fast cancellation. Forgetting to call any
	// of acquire/release/recover/Done is a bug; on this hand-written
	// path the developer is responsible for getting all of them right.
	runRequired := func(label string, fn func(ctx context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					failRequired(fmt.Errorf("panic in %s: %v\n%s", label, r, debug.Stack()))
				}
			}()
			if err := acquire(ctx); err != nil {
				failRequired(err)
				return
			}
			defer release()
			if err := fn(ctx); err != nil {
				failRequired(err)
			}
		}()
	}

	// runOptional wraps an optional-field task. Errors are swallowed
	// (caller is expected to detect them via the result-field default).
	// Panics are also swallowed but logged via the same path.
	runOptional := func(label string, fn func(ctx context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			if err := acquire(ctx); err != nil {
				return
			}
			defer release()
			_ = fn(ctx)
			_ = label // would log here in real code
		}()
	}

	// Required: profile, posts, subscription.
	runRequired("profile", func(ctx context.Context) error {
		p, err := a.B.FetchProfile(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		profile = p
		mu.Unlock()
		return nil
	})
	runRequired("posts", func(ctx context.Context) error {
		ps, err := a.B.FetchPosts(ctx, cfg.UserID, cfg.PostLimit)
		if err != nil {
			return err
		}
		mu.Lock()
		posts = ps
		mu.Unlock()
		return nil
	})
	runRequired("subscription", func(ctx context.Context) error {
		s, err := a.B.FetchSubscription(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		subscription = s
		mu.Unlock()
		return nil
	})

	// Optional: recommendations, notifications.
	runOptional("recommendations", func(ctx context.Context) error {
		rs, err := a.B.FetchRecommendations(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		recs = rs
		mu.Unlock()
		return nil
	})
	runOptional("notifications", func(ctx context.Context) error {
		n, err := a.B.FetchNotificationCount(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		notifCount = n
		mu.Unlock()
		return nil
	})

	wg.Wait()
	if requiredErr != nil {
		return Dashboard{}, requiredErr
	}

	// Sub-fan-out: enrich each post with its author name. We must NOT
	// reuse the outer waitgroup because it is already drained, and we
	// must NOT reuse the outer cancel because the SLA deadline is still
	// active. We start a fresh nested fan-out with a separate wg + mu.
	enriched := make([]backends.EnrichedPost, len(posts))
	var ewg sync.WaitGroup
	var emu sync.Mutex
	var enrichErr error
	failEnrich := func(err error) {
		emu.Lock()
		if enrichErr == nil {
			enrichErr = err
			cancel()
		}
		emu.Unlock()
	}
	for i, p := range posts {
		i, p := i, p
		ewg.Add(1)
		go func() {
			defer ewg.Done()
			defer func() {
				if r := recover(); r != nil {
					failEnrich(fmt.Errorf("panic in author: %v", r))
				}
			}()
			if err := acquire(ctx); err != nil {
				failEnrich(err)
				return
			}
			defer release()
			name, err := a.B.FetchAuthorName(ctx, p.AuthorID)
			if err != nil {
				failEnrich(err)
				return
			}
			emu.Lock()
			enriched[i] = backends.EnrichedPost{Post: p, AuthorName: name}
			emu.Unlock()
		}()
	}
	ewg.Wait()
	if enrichErr != nil {
		return Dashboard{}, enrichErr
	}

	return Dashboard{
		Profile:           profile,
		Posts:             enriched,
		Subscription:      subscription,
		Recommendations:   recs,
		NotificationCount: notifCount,
	}, nil
}

// Compile-time check that BareGoAggregator implements Aggregator.
var _ Aggregator = (*BareGoAggregator)(nil)

// errAggregateBare is unused; kept to silence import grouping if needed.
var _ = errors.New
