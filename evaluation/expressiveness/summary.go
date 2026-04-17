package expressiveness

// Expressiveness comparison summary table.
//
// ┌───────────────┬──────────┬───────────┬──────────┬────────┬──────┐
// │ Scenario      │ Approach │ SLOC      │ SYNC     │ CANCEL │ BUGS │
// ├───────────────┼──────────┼───────────┼──────────┼────────┼──────┤
// │ EX-1: Happy   │ bare     │ 11        │ 1        │ 0      │ 2    │
// │               │ errgroup │  8        │ 0        │ 0      │ 1    │
// │               │ scope    │  7        │ 0        │ 0      │ 0    │
// ├───────────────┼──────────┼───────────┼──────────┼────────┼──────┤
// │ EX-2: FailFast│ bare     │ 19        │ 3        │ 2      │ 4    │
// │               │ errgroup │  8        │ 0        │ 0      │ 1    │
// │               │ scope    │  7        │ 0        │ 0      │ 0    │
// ├───────────────┼──────────┼───────────┼──────────┼────────┼──────┤
// │ EX-3: Superv. │ bare     │ 17        │ 2        │ 0      │ 3    │
// │               │ errgroup │  8 (*)    │ 0        │ 0      │ N/A  │
// │               │ scope    │  7        │ 0        │ 0      │ 0    │
// ├───────────────┼──────────┼───────────┼──────────┼────────┼──────┤
// │ EX-4: Nested  │ bare     │ 30        │ 3        │ 3      │ 6    │
// │               │ errgroup │ N/A       │ -        │ -      │ -    │
// │               │ scope    │ 10        │ 0        │ 0      │ 0    │
// └───────────────┴──────────┴───────────┴──────────┴────────┴──────┘
//
// (*) errgroup cannot aggregate errors — marked as fundamentally limited.
// errgroup has no concept of hierarchy — EX-4 is not expressible.
//
// SLOC: source lines of code (non-blank, non-comment, function body only).
// SYNC: manual sync primitives (WaitGroup, Mutex, Once, atomic, channel-as-semaphore).
// CANCEL: explicit cancel()/Done() calls the developer must write.
// BUGS: sites where forgetting a call introduces a bug.
