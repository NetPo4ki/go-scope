// Package main points to the HTTP fan-out demo in examples/fanout.
package main

import "fmt"

func main() {
	fmt.Println("Fan-out demo: run from the repository root:")
	fmt.Println("  go run ./examples/fanout")
	fmt.Println("Then: curl -s http://127.0.0.1:8080/page")
}
