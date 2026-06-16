package example_test

// The tests in this file exercise every implementation of Aggregator
// (BareGo, Errgroup, Conc, Scope) against the same scenarios:
//
//   - happy path: all backends succeed, Dashboard is fully populated;
//   - fail-fast on a required field: aggregator returns an error and
//     stops issuing further calls;
//   - supervisor on an optional field: aggregator succeeds, the optional
//     field carries its sentinel "missing" value;
//   - panic safety: a backend that panics does not crash the process;
//   - SLA: backends slower than the configured SLA cause a deadline error.
//
// All four implementations must behave the same on these tests; this is
// what makes the LOC and bug-site comparison in Chapter 5 fair.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NetPo4ki/go-scope/evaluation/example"
	"github.com/NetPo4ki/go-scope/evaluation/example/backends"
)

// implementations returns one fresh instance of each aggregator wired up
// to the given Backends. Tests iterate over this slice.
func implementations(b *backends.Backends) []struct {
	Name string
	Agg  example.Aggregator
} {
	return []struct {
		Name string
		Agg  example.Aggregator
	}{
		{"BareGo", &example.BareGoAggregator{B: b}},
		{"Errgroup", &example.ErrgroupAggregator{B: b}},
		{"Conc", &example.ConcAggregator{B: b}},
		{"Scope", &example.ScopeAggregator{B: b}},
	}
}

func defaultConfig() example.Config {
	return example.Config{
		UserID:    "u1",
		PostLimit: 5,
		SLA:       500 * time.Millisecond,
		MaxConc:   8,
	}
}

func TestHappyPath(t *testing.T) {
	for _, impl := range implementations(backends.New(0)) {
		t.Run(impl.Name, func(t *testing.T) {
			d, err := impl.Agg.Aggregate(context.Background(), defaultConfig())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Profile.ID != "u1" {
				t.Errorf("Profile.ID = %q, want u1", d.Profile.ID)
			}
			if len(d.Posts) != 5 {
				t.Errorf("len(Posts) = %d, want 5", len(d.Posts))
			}
			for i, p := range d.Posts {
				if p.AuthorName == "" {
					t.Errorf("Posts[%d] missing AuthorName", i)
				}
			}
			if d.Subscription.Tier != "premium" {
				t.Errorf("Subscription.Tier = %q, want premium", d.Subscription.Tier)
			}
			if len(d.Recommendations) == 0 {
				t.Errorf("Recommendations is empty")
			}
			if d.NotificationCount != 7 {
				t.Errorf("NotificationCount = %d, want 7", d.NotificationCount)
			}
		})
	}
}

func TestFailFastOnRequired(t *testing.T) {
	for _, impl := range implementations(makeBackendsWithError(backends.MethodProfile)) {
		t.Run(impl.Name, func(t *testing.T) {
			_, err := impl.Agg.Aggregate(context.Background(), defaultConfig())
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, backends.ErrInjected) && !containsInjected(err) {
				t.Errorf("expected ErrInjected, got %v", err)
			}
		})
	}
}

func TestSupervisorOnOptional(t *testing.T) {
	// Recommendations fails; aggregator must still succeed with nil recs.
	for _, impl := range implementations(makeBackendsWithError(backends.MethodRecommendations)) {
		t.Run(impl.Name, func(t *testing.T) {
			d, err := impl.Agg.Aggregate(context.Background(), defaultConfig())
			if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if d.Recommendations != nil {
				t.Errorf("expected nil Recommendations, got %v", d.Recommendations)
			}
			if d.NotificationCount != 7 {
				t.Errorf("NotificationCount = %d, want 7", d.NotificationCount)
			}
		})
	}
}

func TestPanicSafety(t *testing.T) {
	// A panic in an optional field must not crash the process.
	b := backends.New(0)
	b.Panics[backends.MethodRecommendations] = "boom"
	for _, impl := range implementations(b) {
		t.Run(impl.Name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("aggregator propagated panic: %v", r)
				}
			}()
			d, err := impl.Agg.Aggregate(context.Background(), defaultConfig())
			// Errgroup, Conc, and Scope should all succeed without
			// recommendations. BareGo should also succeed because its
			// runOptional helper recovers panics.
			if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if d.Recommendations != nil {
				t.Errorf("expected nil Recommendations, got %v", d.Recommendations)
			}
		})
	}
}

func TestSLAExceeded(t *testing.T) {
	b := backends.New(0)
	b.Latency[backends.MethodPosts] = 200 * time.Millisecond
	cfg := defaultConfig()
	cfg.SLA = 50 * time.Millisecond
	for _, impl := range implementations(b) {
		t.Run(impl.Name, func(t *testing.T) {
			start := time.Now()
			_, err := impl.Agg.Aggregate(context.Background(), cfg)
			elapsed := time.Since(start)
			if err == nil {
				t.Fatal("expected deadline error, got nil")
			}
			if elapsed > 150*time.Millisecond {
				t.Errorf("aggregator did not respect SLA: took %v", elapsed)
			}
		})
	}
}

// makeBackendsWithError returns a Backends that fails the given method.
func makeBackendsWithError(method string) *backends.Backends {
	b := backends.New(0)
	b.Errors[method] = backends.ErrInjected
	return b
}

// containsInjected walks an error chain looking for ErrInjected. Errgroup
// wraps the error in a context-cancel error when the deadline fires; this
// helper looks past that.
func containsInjected(err error) bool {
	for err != nil {
		if errors.Is(err, backends.ErrInjected) {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
