GO = go

.PHONY: all build test race lint ci bench bench-micro bench-macro bench-full examples-build

all: build test

# Local parity with .github/workflows/ci.yml (minus golangci-lint).
ci: vet
	$(GO) build ./...
	$(GO) test -race ./...
	$(GO) build -o /dev/null ./examples/... ./cmd/...

build:
	$(GO) build ./...

examples-build:
	$(GO) build -o /dev/null ./examples/... ./cmd/...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

lint:
	golangci-lint run

# Quick local benchmark (no high iteration count)
bench:
	$(GO) test -bench=. -benchmem -count=1 ./benchmarks/micro/... ./benchmarks/macro/...

bench-micro:
	$(GO) test -bench=. -benchmem -count=5 ./benchmarks/micro/...

bench-macro:
	$(GO) test -bench=. -benchmem -count=5 ./benchmarks/macro/...

# Full reproducible run (see benchmarks/README.md and scripts/bench.sh)
bench-full:
	./scripts/bench.sh
