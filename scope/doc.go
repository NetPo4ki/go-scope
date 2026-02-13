// Package scope provides structured-concurrency primitives for Go.
// Scopes own the tasks they spawn, provide a join point (Wait), and
// propagate cancellation and errors predictably according to a policy.
package scope


