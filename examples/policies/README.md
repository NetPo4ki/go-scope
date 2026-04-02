# policies

```bash
go run .
```

**Expect:** `FailFast` / errgroup / bare: sibling aborted after `boom`. `Supervisor`: both tasks run; `Wait` returns an aggregated error containing the failure while the other task still finishes.
