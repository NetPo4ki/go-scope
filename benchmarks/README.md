# Benchmarks

- **`micro/`** — spawn/wait overhead and fail-fast vs `x/sync/errgroup` baselines.
- **`macro/`** — synthetic fan-out style workloads (injected error after parallel work).

## Quick run

```bash
go test -bench=. -benchmem ./benchmarks/micro/... ./benchmarks/macro/...
```

## Reproducible run (thesis / CI artifact)

```bash
chmod +x scripts/bench.sh   # once
./scripts/bench.sh
```

Outputs a timestamped log under `benchmarks/results/` (gitignored). Override iterations:

```bash
COUNT=30 RUN_ID=my-run ./scripts/bench.sh
```

Compare runs with [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat benchmarks/results/bench-a.txt benchmarks/results/bench-b.txt
```

Record `go version`, CPU model, and `GOMAXPROCS` in thesis methodology; the script logs `go version` and sets `GOMAXPROCS` if unset.
