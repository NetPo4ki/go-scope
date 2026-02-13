// Package main demonstrates an HTTP fan-out example using scoped concurrency.
package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/NetPo4ki/go-scope/scope"
)

// Dummy types and services for demonstration purposes.
type Profile struct{ ID string }
type Event struct{ Kind string }
type Item struct{ Name string }

func userID(_ *http.Request) string { return "u-123" }

func fetchProfile(ctx context.Context, uid string) (Profile, error) {
	select {
	case <-time.After(30 * time.Millisecond):
		return Profile{ID: uid}, nil
	case <-ctx.Done():
		return Profile{}, ctx.Err()
	}
}

func fetchSearchHistory(ctx context.Context, _ string) ([]Event, error) {
	select {
	case <-time.After(40 * time.Millisecond):
		return []Event{{Kind: "q"}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func categoriesFor(_ string) []string { return []string{"news", "music", "sports", "tech"} }

func fetchRecommendations(ctx context.Context, _, cat string) ([]Item, error) {
	select {
	case <-time.After(60 * time.Millisecond):
		return []Item{{Name: "rec-" + cat}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func render(w http.ResponseWriter, p Profile, hist []Event, recs []Item) {
	_, _ = fmt.Fprintf(w, "profile=%s hist=%d recs=%d\n", p.ID, len(hist), len(recs))
}

// GetPage demonstrates:
// - request-scoped timeout (200ms)
// - fail-fast policy for critical subrequests (profile, history)
// - supervisor child scope for recommendations
// - bounded parallelism for slow path via a simple semaphore
func GetPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := scope.New(ctx, scope.FailFast, scope.WithTimeout(200*time.Millisecond))

	var prof Profile
	s.Go(func(ctx context.Context) error {
		p, err := fetchProfile(ctx, userID(r))
		if err != nil {
			return err
		}
		prof = p
		return nil
	})

	var hist []Event
	s.Go(func(ctx context.Context) error {
		h, err := fetchSearchHistory(ctx, userID(r))
		if err != nil {
			return err
		}
		hist = h
		return nil
	})

	// Recommendations: supervisor scope + bounded concurrency.
	recScope := s.Child(scope.Supervisor)
	sem := make(chan struct{}, 10) // simple semaphore
	var recsMu sync.Mutex
	var recs []Item

	for _, cat := range categoriesFor(userID(r)) {
		cat := cat
		recScope.Go(func(ctx context.Context) error {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			defer func() { <-sem }()
			items, err := fetchRecommendations(ctx, userID(r), cat)
			if err != nil {
				return err
			}
			recsMu.Lock()
			recs = append(recs, items...)
			recsMu.Unlock()
			return nil
		})
	}

	_ = recScope.Wait()
	if err := s.Wait(); err != nil {
		http.Error(w, "degraded: "+err.Error(), http.StatusPartialContent)
		return
	}
	render(w, prof, hist, recs)
}

func main() {
	// Minimal HTTP server to demo the flow.
	http.HandleFunc("/page", GetPage)
	_ = http.ListenAndServe(":8080", nil)
}
