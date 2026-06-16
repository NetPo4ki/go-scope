// Package backends provides deterministic in-process mock implementations of
// the downstream services used by the user-dashboard aggregator example.
//
// The aggregator scenario models a typical Backend-for-Frontend / GraphQL
// resolver pattern: a single HTTP handler fans out to several internal
// services to assemble one client response. Real production code in this
// pattern is the dominant source of the leaked-goroutine and lost-error
// bugs reported in the Go bug studies cited in the thesis.
//
// Backends supports per-method latency injection, error injection, and
// panic injection so that the same test/benchmark code can exercise
// happy-path, partial-failure, fail-fast, panic-safety, and timeout
// behaviour deterministically without a real network.
package backends

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// Profile is the user's account-level information.
type Profile struct {
	ID     string
	Name   string
	Email  string
	Avatar string
}

// Post is a single user post (without enrichment).
type Post struct {
	ID       string
	AuthorID string
	Title    string
	Body     string
}

// EnrichedPost is a Post plus the author's display name, which the dashboard
// needs to render. The author name is fetched as a sub-fan-out per post and
// is the reason the example needs hierarchical scope ownership.
type EnrichedPost struct {
	Post
	AuthorName string
}

// Recommendation is an optional content recommendation.
type Recommendation struct {
	ID    string
	Title string
}

// Subscription is the user's billing tier and expiration.
type Subscription struct {
	Tier      string
	ExpiresAt time.Time
}

// Method names used to address per-call injection.
const (
	MethodProfile           = "profile"
	MethodPosts             = "posts"
	MethodAuthor            = "author"
	MethodRecommendations   = "recommendations"
	MethodNotificationCount = "notifications"
	MethodSubscription      = "subscription"
)

// Backends is a deterministic mock cluster of downstream services used by
// every aggregator implementation. All methods respect context cancellation,
// matching the contract of a properly-written Go RPC client.
//
// The zero value is a healthy cluster: every call returns canned data with
// the configured BaseLatency. Tests configure Latency, Errors, or Panics
// to inject specific failure scenarios.
type Backends struct {
	// BaseLatency is added to every call to simulate a wire round-trip.
	BaseLatency time.Duration

	// Latency overrides BaseLatency for a specific method when non-zero.
	// Key = method name (one of the Method* constants).
	Latency map[string]time.Duration

	// Errors causes a method to return the given error after its latency
	// elapses. Key = method name.
	Errors map[string]error

	// Panics causes a method to panic with the given value after its
	// latency elapses. Used for the panic-safety tests.
	// Key = method name.
	Panics map[string]string

	// Calls counts invocations of each method (atomic). Used by tests to
	// verify that fail-fast cancellation actually stopped subsequent
	// calls.
	Calls map[string]*atomic.Int64
}

// New returns a healthy Backends with the given base latency. Latency,
// Errors, and Panics maps are initialised empty and may be assigned to
// directly by tests.
func New(baseLatency time.Duration) *Backends {
	return &Backends{
		BaseLatency: baseLatency,
		Latency:     map[string]time.Duration{},
		Errors:      map[string]error{},
		Panics:      map[string]string{},
		Calls: map[string]*atomic.Int64{
			MethodProfile:           {},
			MethodPosts:             {},
			MethodAuthor:            {},
			MethodRecommendations:   {},
			MethodNotificationCount: {},
			MethodSubscription:      {},
		},
	}
}

// wait delays for the configured latency, returning ctx.Err() if cancelled
// during the delay.
func (b *Backends) wait(ctx context.Context, method string) error {
	d := b.BaseLatency
	if v, ok := b.Latency[method]; ok {
		d = v
	}
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// dispatch is the common path for every method: count, wait, optionally
// panic, optionally return an injected error.
func (b *Backends) dispatch(ctx context.Context, method string) error {
	if c := b.Calls[method]; c != nil {
		c.Add(1)
	}
	if err := b.wait(ctx, method); err != nil {
		return err
	}
	if msg, ok := b.Panics[method]; ok {
		panic(msg)
	}
	if err, ok := b.Errors[method]; ok {
		return err
	}
	return nil
}

// FetchProfile returns the user's profile or ctx.Err() / injected error.
func (b *Backends) FetchProfile(ctx context.Context, userID string) (Profile, error) {
	if err := b.dispatch(ctx, MethodProfile); err != nil {
		return Profile{}, err
	}
	return Profile{
		ID:     userID,
		Name:   "User " + userID,
		Email:  userID + "@example.com",
		Avatar: "https://cdn.example.com/" + userID + ".png",
	}, nil
}

// FetchPosts returns up to limit recent posts for the user.
func (b *Backends) FetchPosts(ctx context.Context, userID string, limit int) ([]Post, error) {
	if err := b.dispatch(ctx, MethodPosts); err != nil {
		return nil, err
	}
	out := make([]Post, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, Post{
			ID:       fmt.Sprintf("%s-p%d", userID, i),
			AuthorID: userID,
			Title:    fmt.Sprintf("Post %d", i),
			Body:     "...",
		})
	}
	return out, nil
}

// FetchAuthorName returns the display name for an author. Called once per
// post during enrichment; this is the sub-fan-out that motivates scope
// hierarchy.
func (b *Backends) FetchAuthorName(ctx context.Context, authorID string) (string, error) {
	if err := b.dispatch(ctx, MethodAuthor); err != nil {
		return "", err
	}
	return "Author " + authorID, nil
}

// FetchRecommendations returns recommended content; on error the aggregator
// is expected to log and return without recommendations (supervisor policy
// on the optional fields).
func (b *Backends) FetchRecommendations(ctx context.Context, userID string) ([]Recommendation, error) {
	if err := b.dispatch(ctx, MethodRecommendations); err != nil {
		return nil, err
	}
	return []Recommendation{
		{ID: "r1", Title: "Recommended item 1"},
		{ID: "r2", Title: "Recommended item 2"},
	}, nil
}

// FetchNotificationCount returns the unread-notification count; optional.
// On error the aggregator returns the response with NotificationCount = -1.
func (b *Backends) FetchNotificationCount(ctx context.Context, userID string) (int, error) {
	if err := b.dispatch(ctx, MethodNotificationCount); err != nil {
		return 0, err
	}
	return 7, nil
}

// FetchSubscription returns the user's billing tier; required.
// On error the entire response fails.
func (b *Backends) FetchSubscription(ctx context.Context, userID string) (Subscription, error) {
	if err := b.dispatch(ctx, MethodSubscription); err != nil {
		return Subscription{}, err
	}
	return Subscription{
		Tier:      "premium",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}, nil
}

// ErrInjected is the sentinel error tests inject via Backends.Errors.
var ErrInjected = errors.New("backends: injected failure")
