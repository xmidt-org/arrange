package arrange

import (
	"reflect"
)

// errorType is just the cached reflection lookup for the error type
var errorType reflect.Type = reflect.TypeOf((*error)(nil)).Elem()

// ErrorType returns the reflection type for the error interface
func ErrorType() reflect.Type {
	return errorType
}

// NewErrorValue is a convenience for safely producing a reflect.Value from an error.
// Useful when creating function stubs for reflect.MakeFunc.
func NewErrorValue(err error) reflect.Value {
	errPtr := reflect.New(ErrorType())
	if err != nil {
		errPtr.Elem().Set(reflect.ValueOf(err))
	}

	return errPtr.Elem()
}

// Target describes a sink for an unmarshal operation
type Target struct {
	component   reflect.Value
	unmarshalTo reflect.Value
}

func NewTarget(prototype interface{}) (t Target) {
	pvalue := reflect.ValueOf(prototype)
	if pvalue.Kind() == reflect.Ptr {
		t.unmarshalTo = reflect.New(pvalue.Type().Elem())
		if !pvalue.IsNil() {
			t.unmarshalTo.Elem().Set(pvalue.Elem())
		}

		t.component = t.unmarshalTo
	} else {
		t.unmarshalTo = reflect.New(pvalue.Type())
		t.unmarshalTo.Elem().Set(pvalue)
		t.component = t.unmarshalTo.Elem()
	}

	return
}

// Component returns the component object returned from constructors
func (t Target) Component() interface{} {
	return t.component.Interface()
}

// UnmarshalTo returns the target of unmarshaling, which will always be
// a pointer to the object returned by Component.
func (t Target) UnmarshalTo() interface{} {
	return t.unmarshalTo.Interface()
}

// ComponentFunc produces a reflect.Type for the function signature producing
// the component type of this target.  The set of inputs may be anything desired,
// but the return values will always be (component.Type(), error).
func (t Target) ComponentFuncOf(in ...reflect.Type) reflect.Type {
	return reflect.FuncOf(
		// inputs:
		in,

		// outputs:
		[]reflect.Type{t.component.Type(), ErrorType()},

		// we're not variadic:
		false,
	)
}
