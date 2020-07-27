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

// NewTarget reflects a prototype object that describes the target
// for an unmarshaling operation.  The various unmarshalers and providers
// in this package that accept prototype objects use this function.
//
// The prototype itself is somewhat flexible:
//
// (1) The prototype may be a struct value.  A new struct is created with fields
// set to the same values as the prototype prior to unmarshaling.  The component
// will be a struct value of the same type, i.e. not a pointer to a struct.
//
//   NewTarget(Config{})
//   NewTarget(Config{Timeout: 15 * time.Second}) // a default value for Timeout
//
// can be used with:
//
//   fx.New(
//     fx.Invoke(
//       func(cfg Config) {},
//     ),
//   )
//
// (2) The prototype may be a non-nil pointer to a struct.  A new struct will be
// allocated with fields set to the same values as the prototype prior to
// unmarshaling.  The component will be pointer to this new struct.
//
//   NewTarget(&Config{})
//   NewTarget(new(Config))
//   NewTarget(&Config{Timeout: 15 * time.Second}) // a default value for Timeout
//
// can be used with:
//
//   fx.New(
//     fx.Invoke(
//       func(cfg *Config) {
//         // always a non-nil pointer, but any fields not unmarshaled
//         // will be set to their corresponding fields in the prototype
//       },
//     ),
//   )
//
// (3) The prototype may be a nil pointer to a struct.  A new struct of the same type
// will be created, but with all fields set to their zero values prior to unmarshaling.
// The component will be a pointer to this new struct.
//
//   NewTarget((*Config)(nil))
//
// can be used with:
//
//   fx.New(
//     fx.Invoke(
//       func(cfg *Config) {
//         // always a non-nil pointer, but any fields not unmarshaled
//         // will be set to their zero values
//       },
//     ),
//   )
//
// If the prototype does not refer to a struct, the results of this function are undefined.
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

// ComponentType returns the type of the component object, which is
// useful when building function signatures with reflect.FuncOf.
func (t Target) ComponentType() reflect.Type {
	return t.component.Type()
}

// UnmarshalTo returns the target of unmarshaling, which will always be
// a pointer.  If the component is a value, this method returns a pointer
// to that value.  If the component is a pointer, this method returns that
// same pointer.
func (t Target) UnmarshalTo() interface{} {
	return t.unmarshalTo.Interface()
}
