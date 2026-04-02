# zombie

```bash
go run .
```

**Expect:** Three blocks each print `loop terminated` after cancel—goroutine exits via `ctx.Done()` instead of running forever.
