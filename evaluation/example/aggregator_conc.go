package example

import (
	"context"
	"sync"

	"github.com/sourcegraph/conc/pool"

	"github.com/NetPo4ki/go-scope/evaluation/example/backends"
)

// ConcAggregator implements the dashboard handler using sourcegraph/conc.
//
// conc improves on errgroup in two ways that matter to this scenario:
// panic recovery is built into the pool (panics are converted to errors),
// and the concurrency limit (WithMaxGoroutines) is integrated with the
// pool's lifecycle. The application code therefore drops the manual
// recoverWrap that errgroup needs.
//
// conc does not improve on errgroup for the supervisor / fail-fast
// distinction at the level of individual fields: a pool's policy is fixed
// at construction (WithFirstError = fail-fast, no WithFirstError =
// collect all), so required fields and optional fields still have to live
// in separate pools and the optional pool still has to swallow errors at
// the task boundary.
//
// conc has no notion of hierarchy. The author-enrichment sub-fan-out is
// therefore again a fresh top-level pool, sharing nothing with the parent.
type ConcAggregator struct {
	B *backends.Backends
}

func (a *ConcAggregator) Aggregate(ctx context.Context, cfg Config) (Dashboard, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.SLA)
	defer cancel()

	// Required-fields pool: fail-fast on first error, panic-safe by
	// default, max-conc bounded.
	pReq := pool.New().
		WithContext(ctx).
		WithMaxGoroutines(cfg.MaxConc).
		WithCancelOnError().
		WithFirstError()

	var (
		mu           sync.Mutex
		profile      backends.Profile
		posts        []backends.Post
		subscription backends.Subscription
	)

	pReq.Go(func(ctx context.Context) error {
		p, err := a.B.FetchProfile(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		profile = p
		mu.Unlock()
		return nil
	})
	pReq.Go(func(ctx context.Context) error {
		ps, err := a.B.FetchPosts(ctx, cfg.UserID, cfg.PostLimit)
		if err != nil {
			return err
		}
		mu.Lock()
		posts = ps
		mu.Unlock()
		return nil
	})
	pReq.Go(func(ctx context.Context) error {
		s, err := a.B.FetchSubscription(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		subscription = s
		mu.Unlock()
		return nil
	})

	// Optional-fields pool. Two extra concerns the developer must handle
	// manually here:
	//
	//   1. conc.ContextPool has no per-task supervisor flag, so each
	//      task swallows its own error by always returning nil to
	//      prevent the pool from cancelling siblings.
	//   2. conc.ContextPool's panic handling is "catch in worker,
	//      re-panic in Wait()", which would propagate a panicking
	//      optional task all the way up. We therefore wrap every
	//      optional task body in defer/recover, the same as in the
	//      bare-Go implementation.
	pOpt := pool.New().
		WithContext(ctx).
		WithMaxGoroutines(cfg.MaxConc)

	var recs []backends.Recommendation
	notifCount := -1

	pOpt.Go(func(ctx context.Context) error {
		defer func() { _ = recover() }()
		rs, err := a.B.FetchRecommendations(ctx, cfg.UserID)
		if err == nil {
			mu.Lock()
			recs = rs
			mu.Unlock()
		}
		return nil
	})
	pOpt.Go(func(ctx context.Context) error {
		defer func() { _ = recover() }()
		n, err := a.B.FetchNotificationCount(ctx, cfg.UserID)
		if err == nil {
			mu.Lock()
			notifCount = n
			mu.Unlock()
		}
		return nil
	})

	if err := pReq.Wait(); err != nil {
		cancel()
		_ = pOpt.Wait()
		return Dashboard{}, err
	}
	_ = pOpt.Wait()

	// Author-enrichment sub-fan-out: a third top-level pool, since conc
	// has no parent-child scopes. The MaxConcurrency configured on each
	// pool is independent, so the total in-flight count over the
	// lifetime of the request can briefly exceed cfg.MaxConc.
	pEn := pool.New().
		WithContext(ctx).
		WithMaxGoroutines(cfg.MaxConc).
		WithCancelOnError().
		WithFirstError()
	enriched := make([]backends.EnrichedPost, len(posts))
	for i, p := range posts {
		i, p := i, p
		pEn.Go(func(ctx context.Context) error {
			name, err := a.B.FetchAuthorName(ctx, p.AuthorID)
			if err != nil {
				return err
			}
			enriched[i] = backends.EnrichedPost{Post: p, AuthorName: name}
			return nil
		})
	}
	if err := pEn.Wait(); err != nil {
		return Dashboard{}, err
	}

	return Dashboard{
		Profile:           profile,
		Posts:             enriched,
		Subscription:      subscription,
		Recommendations:   recs,
		NotificationCount: notifCount,
	}, nil
}

var _ Aggregator = (*ConcAggregator)(nil)
