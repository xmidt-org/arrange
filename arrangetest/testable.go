package arrangetest

import (
	"fmt"
	"testing"
)

// Testable is the minimal interface required for assertions and testing.
// This interface is implemented by several libraries.
type Testable interface {
	Logf(string, ...interface{})
	Errorf(string, ...interface{})
	FailNow()
}

// AsTestable converts a value into a Testable.  The v parameter
// may be a *testing.T, *testing.B, or a type that provides a T() *testing.T method.
//
// If v cannot be coerced into a Testable, this function panics.
func AsTestable(v any) Testable {
	if tt, ok := v.(Testable); ok {
		return tt
	}

	type testHolder interface {
		T() *testing.T
	}

	if th, ok := v.(testHolder); ok {
		return th.T()
	}

	panic(fmt.Errorf("%T cannot be converted into a Testable", v))
}
