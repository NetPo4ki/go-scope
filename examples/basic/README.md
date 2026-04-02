# basic

```bash
go run .
```

**Expect:** The fast task returns an error (~50ms); FailFast cancels the slow task (`task1 canceled`). Same story for `errgroup` and the bare `sync.Once` + `cancel` pattern.
