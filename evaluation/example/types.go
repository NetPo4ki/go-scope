// Package example contains four implementations of the same realistic
// HTTP-handler aggregator workload:
//
//	BareGo     — manual sync.WaitGroup + sync.Mutex coordination
//	Errgroup   — golang.org/x/sync/errgroup
//	Conc       — github.com/sourcegraph/conc/pool
//	Scope      — github.com/NetPo4ki/go-scope/scope (this thesis)
//
// Each implementation honours the same external contract (Aggregate(...)
// returns a Dashboard or an error) and exhibits the same observable
// behaviour for happy-path inputs. They diverge on:
//
//   - whether a panic in a backend handler crashes the process,
//   - whether failure of an optional field fails the whole request,
//   - how cancellation propagates to in-flight backend calls,
//   - how concurrency is bounded against the downstream services,
//   - how much application code is required to express the policy.
//
// The implementations are kept as close to idiomatic for their respective
// libraries as possible, so that the comparison reflects the libraries'
// design choices rather than a particular author's style.
package example

import (
	"context"
	"time"

	"github.com/NetPo4ki/go-scope/evaluation/example/backends"
)

// Dashboard is the response of every aggregator implementation.
//
// Required fields (Profile, Posts, Subscription) must be populated for the
// response to be considered successful. If any required-field fetch fails,
// the aggregator returns the error and discards partial results.
//
// Optional fields (Recommendations, NotificationCount) follow a supervisor
// policy: their failure is logged and the response is returned without
// them. NotificationCount = -1 indicates the optional fetch failed.
type Dashboard struct {
	Profile           backends.Profile
	Posts             []backends.EnrichedPost
	Subscription      backends.Subscription
	Recommendations   []backends.Recommendation // nil if optional fetch failed
	NotificationCount int                       // -1 if optional fetch failed
}

// Config holds the parameters every aggregator uses. The same Config is
// passed to every implementation to keep the comparison fair.
type Config struct {
	UserID    string
	PostLimit int           // top-N posts to fetch
	SLA       time.Duration // total-handler deadline (e.g. 200ms)
	MaxConc   int           // upper bound on concurrent outbound RPCs
}

// Aggregator is the type returned by every implementation's constructor.
// It exists so that benchmarks and correctness tests can iterate over a
// uniform set of implementations without type-switching.
type Aggregator interface {
	Aggregate(ctx context.Context, cfg Config) (Dashboard, error)
}
