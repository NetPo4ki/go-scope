package example

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/NetPo4ki/go-scope/evaluation/example/backends"
)

// ErrgroupAggregator implements the dashboard handler using
// golang.org/x/sync/errgroup, the de-facto standard Go fan-out helper.
//
// errgroup gives us three things compared to BareGoAggregator: it owns the
// WaitGroup, it tracks the first error, and (via WithContext) it cancels
// the group context on the first error. Everything else still has to be
// hand-written: panic recovery (errgroup does not provide it), the
// supervisor policy for optional fields (errgroup does not have separate
// policies, so optional fields cannot share a single group with required
// fields), the concurrency limiter (errgroup.SetLimit exists but is not
// cancellation-aware), and the second-tier sub-fan-out for author
// enrichment (a new errgroup per stage).
type ErrgroupAggregator struct {
	B *backends.Backends
}

func (a *ErrgroupAggregator) Aggregate(ctx context.Context, cfg Config) (Dashboard, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.SLA)
	defer cancel()

	// Required-fields group: fail-fast via errgroup.WithContext.
	gReq, ctxReq := errgroup.WithContext(ctx)
	gReq.SetLimit(cfg.MaxConc)

	var (
		mu           sync.Mutex
		profile      backends.Profile
		posts        []backends.Post
		subscription backends.Subscription
	)

	// recoverWrap is required because errgroup does not recover panics.
	// Forgetting it crashes the process.
	recoverWrap := func(label string, fn func() error) func() error {
		return func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in %s: %v\n%s", label, r, debug.Stack())
				}
			}()
			return fn()
		}
	}

	gReq.Go(recoverWrap("profile", func() error {
		p, err := a.B.FetchProfile(ctxReq, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		profile = p
		mu.Unlock()
		return nil
	}))
	gReq.Go(recoverWrap("posts", func() error {
		ps, err := a.B.FetchPosts(ctxReq, cfg.UserID, cfg.PostLimit)
		if err != nil {
			return err
		}
		mu.Lock()
		posts = ps
		mu.Unlock()
		return nil
	}))
	gReq.Go(recoverWrap("subscription", func() error {
		s, err := a.B.FetchSubscription(ctxReq, cfg.UserID)
		if err != nil {
			return err
		}
		mu.Lock()
		subscription = s
		mu.Unlock()
		return nil
	}))

	// Optional-fields group: errors must be swallowed because errgroup
	// fails the whole group on any returned error and offers no way to
	// distinguish required from optional. We therefore run optional
	// fields in a separate errgroup that always returns nil from each
	// task, recording results into shared variables under a mutex.
	gOpt, ctxOpt := errgroup.WithContext(ctx)
	gOpt.SetLimit(cfg.MaxConc)

	var recs []backends.Recommendation
	notifCount := -1

	gOpt.Go(recoverWrap("recommendations", func() error {
		rs, err := a.B.FetchRecommendations(ctxOpt, cfg.UserID)
		if err == nil {
			mu.Lock()
			recs = rs
			mu.Unlock()
		}
		return nil // never fail the optional group
	}))
	gOpt.Go(recoverWrap("notifications", func() error {
		n, err := a.B.FetchNotificationCount(ctxOpt, cfg.UserID)
		if err == nil {
			mu.Lock()
			notifCount = n
			mu.Unlock()
		}
		return nil
	}))

	// Wait for required first; if it fails, cancel the optional group.
	if err := gReq.Wait(); err != nil {
		cancel() // also cancels ctxOpt via the parent ctx
		_ = gOpt.Wait()
		return Dashboard{}, err
	}
	_ = gOpt.Wait()

	// Sub-fan-out: author enrichment. We need a third errgroup for the
	// per-post fan-out because each errgroup is single-shot (Wait drains
	// it). Note we deliberately do not share the limiter across stages,
	// which means the total in-flight count can exceed MaxConc. errgroup
	// has no concept of a multi-stage shared limiter.
	gEn, ctxEn := errgroup.WithContext(ctx)
	gEn.SetLimit(cfg.MaxConc)
	enriched := make([]backends.EnrichedPost, len(posts))
	for i, p := range posts {
		i, p := i, p
		gEn.Go(recoverWrap("author", func() error {
			name, err := a.B.FetchAuthorName(ctxEn, p.AuthorID)
			if err != nil {
				return err
			}
			enriched[i] = backends.EnrichedPost{Post: p, AuthorName: name}
			return nil
		}))
	}
	if err := gEn.Wait(); err != nil {
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

var _ Aggregator = (*ErrgroupAggregator)(nil)
