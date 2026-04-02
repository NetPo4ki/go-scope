GO = go

.PHONY: all build test race lint bench bench-micro bench-macro bench-full

all: build test

build:
	$(GO) build ./...

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
