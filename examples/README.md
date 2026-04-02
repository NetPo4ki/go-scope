# Examples (defense / teaching demos)

Run from repo root with `go run ./examples/<name>`.

| Directory | Idea |
|-----------|------|
| [basic](./basic/) | Minimal FailFast: one error cancels a slow sibling; vs errgroup / bare. |
| [lifecycle](./lifecycle/) | Parent cancel vs child scope; vs errgroup / manual `WaitGroup`. |
| [policies](./policies/) | `FailFast` vs `Supervisor` vs errgroup-only fail-fast. |
| [observability](./observability/) | `scope.Observer` hooks vs manual counting. |
| [zombie](./zombie/) | Long-lived loop exits on cancel (no zombie) in three styles. |
| [fanout](./fanout/) | HTTP handler: timeout, child `Supervisor`, `WithMaxConcurrency`. |

For benchmarks and reproducible runs, see [../benchmarks/README.md](../benchmarks/README.md).
