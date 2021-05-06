package arrange

import (
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

// NotAFunctionError indicates that an element of a Bind was not a function
// and thus couldn't be emitted as an fx.App function option.
type NotAFunctionError struct {
	Type reflect.Type
}

// Error satisfies the error interface.
func (nofe *NotAFunctionError) Error() string {
	return fmt.Sprintf("%s is not a function", nofe.Type)
}

// Bind is a slice of functions that participate in dependency injection inside an fx.App.
// Each function may have its arguments injected from the containing fx.App or supplied via With.
// Any type other than a function will short-circuit fx.App startup with an error.
//
// The motivation for this type is situations where some objects come from outside the fx.App
// and it may be undesirable to supply them as components.  The following two code examples are
// equivalent:
//
//   // standard way of referring to non-components
//   // there is no need for Bind in this case
//   var buffer *bytes.Buffer
//   app := fx.New(
//     fx.Invoke(
//       func() {
//         // just use the variable in scope
//         buffer.WriteString("hello, world")
//       },
//     ),
//   )
//
//   // using Bind.With
//   // assume that internal, unexported code creates buffer
//   var buffer *bytes.Buffer
//   app := fx.New(
//     arrange.Bind{
//       func(buf *bytes.Buffer) {
//         // buf is populated by With
//         buf.WriteString("hello, world")
//       },
//     }.With(buffer),
//   )
//
// The signature for each function is very flexible and can be most anything allowed by
// fx.Provide or fx.Invoke.
type Bind []interface{}

// buildWrapperIn inspects an inner function's input parameters and separates them into
// two categories: (1) those that are dependencies, and (2) those that come from bindings.
// Ultimately, the dependencies will be the input parameters of the wrapping function.
func buildWrapperIn(inner reflect.Type, withValues []reflect.Value) (deps []reflect.Type, bindings []reflect.Value) {
	bindings = make([]reflect.Value, inner.NumIn())
	for i := 0; i < inner.NumIn(); i++ {
		inType := inner.In(i)

		for _, bv := range withValues {
			if bv.Type().AssignableTo(inType) {
				bindings[i] = bv
			}
		}

		if !bindings[i].IsValid() {
			deps = append(deps, inType)
		}
	}

	return
}

// buildWrapperOut creates the slice of output parameter types for the wrapper function.
// This is just a copy of whatever output parameters returned by the inner function.
func buildWrapperOut(inner reflect.Type) (out []reflect.Type) {
	out = make([]reflect.Type, inner.NumOut())
	for i := 0; i < inner.NumOut(); i++ {
		out[i] = inner.Out(i)
	}

	return
}

// wrapperFunc produces a function for use with reflect.MakeFunc that invokes the inner function.
// Any parameter that is bound in bindings will be passed through as is.  Any parameter that is
// not bound is expected to be supplied by dependency injection.
func wrapperFunc(inner reflect.Value, bindings []reflect.Value) func([]reflect.Value) []reflect.Value {
	return func(args []reflect.Value) []reflect.Value {
		argp := 0
		parameters := make([]reflect.Value, len(bindings))
		for i, bv := range bindings {
			if bv.IsValid() {
				parameters[i] = bv
			} else {
				parameters[i] = args[argp]
				argp++
			}
		}

		return inner.Call(parameters)
	}
}

// With constructs a sequence of fx.Provide or fx.Invoke options wherein each function
// in this Bind instance has its input parameters supplied by a mixture of the values passed
// to this method with the set of dependencies in the enclosing fx.App.  Any bound value
// passed to this method will override any duplicate dependency of the same type.
//
// Each function that has no output parameters or returns a single output parameter that
// implements error is assumed to be an fx.Invoke function.  Otherwise, fx.Provide is
// used, making that function a constructor whose return values can participate in
// further dependency injection.
//
// If args is empty, each function is supplied to the enclosing fx.App as is, albeit
// as either an fx.Invoke or fx.Provide function dependending on the output parameters.
//
// If this Bind is empty, then the returned fx.Option is a noop.
func (b Bind) With(args ...interface{}) fx.Option {
	withValues := make([]reflect.Value, 0, len(args))
	for _, a := range args {
		withValues = append(withValues, reflect.ValueOf(a))
	}

	var appOpts []fx.Option
	for _, f := range b {
		fv := reflect.ValueOf(f)
		if fv.Kind() != reflect.Func {
			appOpts = append(appOpts, fx.Error(
				&NotAFunctionError{
					Type: fv.Type(),
				},
			))

			continue
		}

		ft := fv.Type()

		// choose either fx.Invoke or fx.Provide, based on the return values
		optFunc := fx.Invoke
		if ft.NumOut() > 1 || (ft.NumOut() == 1 && !ft.Out(0).Implements(ErrorType())) {
			optFunc = fx.Provide
		}

		deps, bindings := buildWrapperIn(ft, withValues)

		// choose the target function based on whether anything was bound
		target := f
		if len(deps) < ft.NumIn() {
			target = reflect.MakeFunc(
				reflect.FuncOf(deps, buildWrapperOut(ft), false), // TODO: handle variadic somehow?
				wrapperFunc(fv, bindings),
			).Interface()
		}

		appOpts = append(appOpts, optFunc(target))
	}

	return fx.Options(appOpts...)
}
