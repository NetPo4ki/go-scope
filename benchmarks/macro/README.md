# Macro benchmarks

Synthetic workloads comparable across `scope` and `x/sync/errgroup`:

- **Fan-out + injected error** — several cooperative “sub-requests” finish quickly, then one task fails; measures drain/cancel behavior under fail-fast.

Run:

```bash
go test -bench=. -benchmem ./benchmarks/macro/...
```

For reproducible multi-run output, use the repo root [`scripts/bench.sh`](../scripts/bench.sh) or `make bench-full`.
