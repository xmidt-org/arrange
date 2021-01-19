package arrange

import (
	"fmt"
	"reflect"

	"go.uber.org/multierr"
)

// InvokeError represents an error on a particular invoke closure
type InvokeError struct {
	Type    reflect.Type
	Message string
}

func (ie *InvokeError) Error() string {
	return fmt.Sprintf("INVOKE ERROR: [%s] %s", ie.Type, ie.Message)
}

// Invoke represents a set of closures with a specific category of signatures.
// Each closure can return exactly 0 or 1 values, where the only value allowed is an error.
//
// The inputs to each closure must all be the same and must match the vector
// of inputs supplied to Call.
//
// The typical use case for an Invoke slice is to wrap it in an fx.Invoke call
// that was generated dynamically.
type Invoke []interface{}

// Call passes the given arguments to each closure in this sequence.  All closures
// must take the same number of arguments.  The reflect package is used to
// convert arguments appropriately, so the types do not need to match exactly as long
// as they're convertible, e.g. an uint32 can be passed to a closure expecting an os.FileMode.
//
// Each closure can either return a single value of type error or return nothing.  All closures
// are invoked, and an aggregate error is returned by this method.
func (ivk Invoke) Call(args ...interface{}) (err error) {
	if len(ivk) == 0 {
		return
	}

	inputs := make([]reflect.Value, 0, len(args))
	for _, f := range ivk {
		fv := ValueOf(f)
		ft := fv.Type()

		switch {
		case fv.Kind() != reflect.Func:
			err = multierr.Append(err, &InvokeError{
				Type:    ft,
				Message: "not a function",
			})

		case ft.NumIn() != len(args):
			err = multierr.Append(err, &InvokeError{
				Type:    ft,
				Message: "wrong number of inputs",
			})

		case ft.NumOut() > 1:
			err = multierr.Append(err, &InvokeError{
				Type:    ft,
				Message: "too many return values",
			})

		case ft.NumOut() == 1 && ft.Out(0) != ErrorType():
			err = multierr.Append(err, &InvokeError{
				Type:    ft,
				Message: "return value is not an error",
			})

		default:
			inputs = inputs[:0]
			for i := 0; i < ft.NumIn(); i++ {
				input := ValueOf(args[i])
				inType := ft.In(i)
				switch {
				case inType == input.Type():
					inputs = append(inputs, input)

				case input.Type().ConvertibleTo(inType):
					inputs = append(inputs, input.Convert(inType))

				default:
					err = multierr.Append(err, &InvokeError{
						Type:    ft,
						Message: fmt.Sprintf("parameter %d is the wrong type", i),
					})
				}
			}

			if len(inputs) == ft.NumIn() {
				outputs := fv.Call(inputs)
				if len(outputs) == 1 && outputs[0].IsValid() && !outputs[0].IsNil() {
					err = multierr.Append(err, outputs[0].Interface().(error))
				}
			}
		}
	}

	return
}
