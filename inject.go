package arrange

import (
	"reflect"
)

// Inject is a slice type intended to hold a sequence of type information
// about injected objects.
//
// This type is particular useful when dynamically building sets of dependencies.
type Inject []interface{}

// Append appends additional injectable types to this instance.  This method
// returns the (possibly) extended Inject instance.
func (ij Inject) Append(more ...interface{}) Inject {
	return append(ij, more...)
}

// Extend appends the contents of several Inject instances and returns the
// (possibly) extended Inject instance.
func (ij Inject) Extend(more ...Inject) Inject {
	for _, e := range more {
		ij = append(ij, e...)
	}

	return ij
}

// Types returns a distinct slice of types suitable for use in building
// dynamic functions such as reflect.FuncOf.  Each element of this slice
// is passed to TypeOf in this package to determine its type.
func (ij Inject) Types() []reflect.Type {
	t := make([]reflect.Type, 0, len(ij))
	for _, v := range ij {
		t = append(t, TypeOf(v))
	}

	return t
}

// FuncOf returns the function signature which takes the types
// defined in this slice as inputs together with the given outputs.
// The returned function type is never variadic.
//
// The returned function type will have this basic signature:
//
//	func(ij[0], ij[1], ... ij[n]) (out[0], out[1], ... out[m])
//
// where n is the length of this Inject instance and m is the length
// of the out variadic slice.
func (ij Inject) FuncOf(out ...reflect.Type) reflect.Type {
	return reflect.FuncOf(
		ij.Types(),
		out,
		false,
	)
}

// MakeFunc wraps a given function that accepts a []reflect.Value as its sole input
// and can return 0 or more outputs.  The fn parameter thus must have this basic
// signature:
//
//	func([]reflect.Value) (out[0], out[1], ... out[m])
//
// The returned function value will have the same set of inputs as returned by FuncOf,
// but will have the set of outputs described by the fn parameter:
//
//	func(ij[0], ij[1], ... ij[n]) (out[0], out[1], ... out[m])
//
// where n is the length of this Inject instance and m is the number of output
// parameters in the fn parameter (which can be zero).
//
// If fn is not a function with the expected sole input parameter of []reflect.Value,
// this method will panic.
//
// This function is useful when dynamically building functions that need to be
// inspected by dependency injection infrastructure.  The wrapper function holds the
// correct signature, and delegates to the fn parameter as its implementation.
func (ij Inject) MakeFunc(fn interface{}) reflect.Value {
	fv := ValueOf(fn)
	ft := fv.Type()

	outTypes := make([]reflect.Type, ft.NumOut())
	for i := 0; i < len(outTypes); i++ {
		outTypes[i] = ft.Out(i)
	}

	return reflect.MakeFunc(
		ij.FuncOf(outTypes...),
		func(inputs []reflect.Value) []reflect.Value {
			// we set up the output parameters using the function itself,
			// so the []reflect.Value results should match
			return fv.Call(
				[]reflect.Value{
					reflect.ValueOf(inputs), // a single input parameter of type []reflect.Value
				},
			)
		},
	)
}
