package scope

import (
	"fmt"
	"runtime/debug"
)

// panicError preserves the panic value and stack trace for diagnostics.
type panicError struct {
	value any
	stack []byte
}

func (e *panicError) Error() string {
	return fmt.Sprintf("panic: %v\n%s", e.value, e.stack)
}

func panicToError(v any) error {
	return &panicError{
		value: v,
		stack: debug.Stack(),
	}
}
