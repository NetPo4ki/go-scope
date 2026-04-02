// Package main points to worker-style workloads in benchmarks.
package main

import "fmt"

func main() {
	fmt.Println("Worker-style synthetic benchmarks:")
	fmt.Println("  go test -bench=BenchmarkFanout -benchmem ./benchmarks/macro/...")
	fmt.Println("Interactive examples:")
	fmt.Println("  go run ./examples/policies")
	fmt.Println("  go run ./examples/zombie")
}
