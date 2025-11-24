GO = go

.PHONY: all build test race lint bench micro

all: build test

build:
	$(GO) build ./...

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

lint:
	golangci-lint run

bench:
	$(GO) test -bench=. -benchtime=1x ./...


