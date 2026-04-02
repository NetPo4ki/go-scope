# observability

```bash
go run .
```

**Expect:** `join latency` line from `ScopeJoined`; counters `started=2 finished=2` for scope. Errgroup and bare blocks show the same counts without lifecycle hooks.

For Prometheus registration, use `observe/prom` `NewExporter` in real services.
