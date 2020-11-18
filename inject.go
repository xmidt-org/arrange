package arrange

import "reflect"

// Inject is a slice type intended to hold a sequence of type information
// about injected objects.  Essentially, an Inject is the set of input parameters
// to an fx.Provide function.
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
func (ij Inject) FuncOf(out ...reflect.Type) reflect.Type {
	return reflect.FuncOf(
		ij.Types(),
		out,
		false,
	)
}

// MakeFunc wraps a given function that accepts a []reflect.Value as its sole input.
// The set of inputs are the same as this Inject sequence.  The wrapped function
// may return any number of output values, which are returned as is by the created function.
//
// The main use case for this method is dynamic creation of fx.Provide constructor
// functions.  Given an application function of the form func([]reflect.Value) (T0, T1, T2...),
// this function produces a wrapper that an enclosing fx.App can inspect to
// determine the correct set of dependencies to inject and the correct set of components
// to emit, if any.
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
