# Evaluation Suite

Comparative evaluation of three concurrency approaches in Go:

| Approach | Description |
|----------|-------------|
| **bare** | `go` + `sync.WaitGroup` + `context.WithCancel` + manual primitives |
| **errgroup** | `golang.org/x/sync/errgroup` — the de-facto community standard |
| **scope** | `github.com/NetPo4ki/go-scope` — structured concurrency with scope-bound lifecycles |

## Structure

```
evaluation/
├── correctness/          # CT-1..CT-5: structural safety demonstrations
├── benchmarks/           # B-1..B-6: quantitative performance comparison
│   └── drain/            # B-2: drain-latency harness (custom main, CSV output)
├── expressiveness/       # EX-1..EX-4: LOC & complexity comparison
├── workload/             # shared work functions (CPU-bound, IO-bound, nop)
└── README.md             # this file
```

## Axes of Comparison

### Axis 1: Correctness (`correctness/`)

Demonstrates classes of bugs that each approach prevents or allows.

| Test | Bug Class | bare | errgroup | scope |
|------|-----------|------|----------|-------|
| CT-1 | Goroutine leak (Go-after-Wait) | Vulnerable | Vulnerable | Protected (TryGo rejects) |
| CT-2 | Panic kills process | Crashes | Crashes | Caught (PanicAsError) |
| CT-3 | Lost errors | Only first | Only first | All aggregated (Supervisor) |
| CT-4 | Orphan child goroutines | No ownership | No hierarchy | Parent joins child |
| CT-5 | Cancel propagation (3 levels) | Manual (3 ctx + 3 wg) | N/A | Automatic |

Run: `go test ./evaluation/correctness/ -v`

### Axis 2: Performance (`benchmarks/`)

| Benchmark | What it measures | Approaches |
|-----------|-----------------|------------|
| B-1: SpawnWait | Pure framework overhead (N=1..1000) | bare, errgroup, scope |
| B-2: Drain | Time from error to full shutdown (percentiles) | bare, errgroup, scope |
| B-3: Supervisor | Error aggregation overhead (N tasks, K errors) | bare, errgroup, scope |
| B-4: Observer | Observability hook cost (no/nop/counting observer) | scope only |
| B-5: Nested | Hierarchical scope overhead (depth 1..3) | bare, scope |
| B-6: Concurrency | Semaphore-bounded execution (N=100, limit=8) | bare, scope |

Run standard benchmarks:
```bash
go test ./evaluation/benchmarks/ -bench=. -benchtime=100x -count=10
```

Run drain-latency harness:
```bash
go run ./evaluation/benchmarks/drain/ -n=100 -samples=200 -out=drain.csv
```

### Axis 3: Expressiveness (`expressiveness/`)

SLOC, manual sync primitives, and bug surface comparison for 4 scenarios.

See `expressiveness/summary.go` for the full table.

### Axis 4: Observability

Qualitative comparison (no code — documented in thesis text).

## Methodology

- **Go version**: from `go.mod` (`go 1.24.0`)
- **GOMAXPROCS**: fixed per run (1, 4, NumCPU)
- **Repetitions**: ≥10 for `benchstat`, ≥200 for drain percentiles
- **Race detector**: `go test -race` on all correctness tests
- **Leak detector**: `go.uber.org/goleak` in core `scope/` tests
- **Reproducibility**: `scripts/bench.sh` sets GOMAXPROCS, logs go version, saves results

## Quick Run

```bash
# Correctness tests
go test ./evaluation/correctness/ -v -race

# All benchmarks (quick, 100 iterations)
go test ./evaluation/benchmarks/ -bench=. -benchtime=100x -count=1

# Full benchmark suite (for thesis, ~5 min)
go test ./evaluation/benchmarks/ -bench=. -benchtime=1000x -count=10 | tee bench.txt

# Drain latency (for thesis)
go run ./evaluation/benchmarks/drain/ -n=100 -samples=500 -out=drain.csv
go run ./evaluation/benchmarks/drain/ -n=1000 -samples=500 -out=drain_1k.csv

# Compare with benchstat
benchstat old.txt new.txt
```
