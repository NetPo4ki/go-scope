package example

import (
	"context"
	"sync"
	"time"

	"github.com/NetPo4ki/go-scope/evaluation/example/backends"
	"github.com/NetPo4ki/go-scope/scope"
)

// ScopeAggregator implements the dashboard handler using go-scope.
//
// The implementation expresses the policy directly:
//
//   - The root FailFast scope holds the SLA timeout, the max-conc
//     limit, and the panic-as-error default. Required fields (profile,
//     posts, subscription) run as direct tasks on the root: any one
//     failing cancels the others.
//   - Optional fields (recommendations, notifications) run on a
//     Supervisor child of the root. A Supervisor child isolates its
//     failures: errors and panics in optional tasks are observable via
//     opt.Wait() but never propagate to the root, so the request never
//     fails because of an optional field.
//   - Author enrichment runs on a second top-level FailFast scope on
//     the original ctx after the first round drains. The new scope
//     inherits the SLA via WithDeadline.
//
// No manual sync.WaitGroup, recover, or sentinel-error channel appears
// in the application code: panic-to-error, policy enforcement, and
// hierarchical isolation are done by the scope.
type ScopeAggregator struct {
	B *backends.Backends
}

func (a *ScopeAggregator) Aggregate(ctx context.Context, cfg Config) (Dashboard, error) {
	root := scope.New(ctx, scope.FailFast,
		scope.WithTimeout(cfg.SLA),
		scope.WithMaxConcurrency(cfg.MaxConc),
	)

	var (
		mu           sync.Mutex
		profile      backends.Profile
		posts        []backends.Post
		subscription backends.Subscription
	)

	root.Go(func(ctx context.Context) error {
		p, err := a.B.FetchProfile(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		profile = p
		mu.Unlock()
		return nil
	})
	root.Go(func(ctx context.Context) error {
		ps, err := a.B.FetchPosts(ctx, cfg.UserID, cfg.PostLimit)
		if err != nil {
			return err
		}
		mu.Lock()
		posts = ps
		mu.Unlock()
		return nil
	})
	root.Go(func(ctx context.Context) error {
		s, err := a.B.FetchSubscription(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		subscription = s
		mu.Unlock()
		return nil
	})

	// Optional fields: Supervisor child of root. Failures here are
	// isolated by the policy and never reach the required-fields scope.
	opt := root.Child(scope.Supervisor)
	var recs []backends.Recommendation
	notifCount := -1
	opt.Go(func(ctx context.Context) error {
		rs, err := a.B.FetchRecommendations(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		recs = rs
		mu.Unlock()
		return nil
	})
	opt.Go(func(ctx context.Context) error {
		n, err := a.B.FetchNotificationCount(ctx, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		notifCount = n
		mu.Unlock()
		return nil
	})

	if err := root.Wait(); err != nil {
		return Dashboard{}, err
	}

	// Author-enrichment phase: a fresh FailFast scope on the original
	// ctx. Because root has drained, we cannot reuse it; the new scope
	// inherits the SLA via WithDeadline.
	enrich := scope.New(ctx, scope.FailFast,
		scope.WithDeadline(deadlineFromContext(ctx, cfg.SLA)),
		scope.WithMaxConcurrency(cfg.MaxConc),
	)
	enriched := make([]backends.EnrichedPost, len(posts))
	for i, p := range posts {
		i, p := i, p
		enrich.Go(func(ctx context.Context) error {
			name, err := a.B.FetchAuthorName(ctx, p.AuthorID)
			if err != nil {
				return err
			}
			enriched[i] = backends.EnrichedPost{Post: p, AuthorName: name}
			return nil
		})
	}
	if err := enrich.Wait(); err != nil {
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

// deadlineFromContext returns the deadline already set on ctx if any,
// otherwise time.Now().Add(fallback).
func deadlineFromContext(ctx context.Context, fallback time.Duration) time.Time {
	if d, ok := ctx.Deadline(); ok {
		return d
	}
	return time.Now().Add(fallback)
}

var _ Aggregator = (*ScopeAggregator)(nil)
