// Package scope provides structured-concurrency primitives for Go.
// Scopes own the tasks they spawn, provide a join point (Wait), and
// propagate cancellation and errors predictably according to a policy.
//
// Lifecycle contract:
//   - Spawn with Go/TryGo while the scope is active.
//   - Join exactly where ownership should end with Wait.
//   - Cancel is idempotent and records the first non-nil cause.
//   - After Cancel or once Wait has started, the scope stops accepting new
//     tasks. Go becomes a no-op, while TryGo reports false.
//   - Parent scopes own child scopes; parent Wait blocks until children finish.
package scope


