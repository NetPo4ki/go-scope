package benchmarks

// B-8: Realistic aggregator end-to-end benchmark.
//
// Each iteration runs one full user-dashboard request against deterministic
// in-process mock backends, with three required and two optional fields,
// followed by a per-post author enrichment fan-out (5 posts). Backend
// latency is set to 100 microseconds per call to mimic an in-region RPC.
//
// The four implementations (bare Go, errgroup, sourcegraph/conc, go-scope)
// are run side by side under identical inputs. The benchmark reports total
// per-request latency including framework overhead, mock-backend latency,
// and dispatch.

import (
	"context"
	"testing"
	"time"

	"github.com/NetPo4ki/go-scope/evaluation/example"
	"github.com/NetPo4ki/go-scope/evaluation/example/backends"
)

// Total backend RPCs per request: profile + posts + subscription +
// recommendations + notifications + 5 author lookups = 10.
//
// At 100us per call and MaxConc=8, the critical path is at least
// ceil(10/8) * 100us = 200us, plus framework overhead.
const aggBenchLatency = 100 * time.Microsecond

func aggBenchConfig() example.Config {
	return example.Config{
		UserID:    "u1",
		PostLimit: 5,
		SLA:       500 * time.Millisecond,
		MaxConc:   8,
	}
}

func benchAgg(b *testing.B, agg example.Aggregator) {
	cfg := aggBenchConfig()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agg.Aggregate(ctx, cfg)
	}
}

func BenchmarkAggregator_Bare(b *testing.B) {
	bks := backends.New(aggBenchLatency)
	benchAgg(b, &example.BareGoAggregator{B: bks})
}

func BenchmarkAggregator_Errgroup(b *testing.B) {
	bks := backends.New(aggBenchLatency)
	benchAgg(b, &example.ErrgroupAggregator{B: bks})
}

func BenchmarkAggregator_Conc(b *testing.B) {
	bks := backends.New(aggBenchLatency)
	benchAgg(b, &example.ConcAggregator{B: bks})
}

func BenchmarkAggregator_Scope(b *testing.B) {
	bks := backends.New(aggBenchLatency)
	benchAgg(b, &example.ScopeAggregator{B: bks})
}
