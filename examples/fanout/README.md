# fanout

```bash
go run .
```

Then open `http://localhost:8080/page` (or `curl localhost:8080/page`).

**Expect:** 200 with body like `profile=u-123 hist=1 recs=4` (four recommendation categories). Root scope uses FailFast + timeout; recommendations run under a `Supervisor` child with `WithMaxConcurrency(10)`.
