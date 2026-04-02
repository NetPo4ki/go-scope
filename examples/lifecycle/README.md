# lifecycle

```bash
go run .
```

**Expect:** After ~50ms, parent cancel stops the child (~150ms work never completes): `scope child: canceled`. Errgroup and bare variants show the same shutdown idea with different wiring.
