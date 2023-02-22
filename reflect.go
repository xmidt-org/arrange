package arrange

import (
	"reflect"

	"go.uber.org/fx"
)

// errorType is the cached reflection lookup for the error type
var errorType reflect.Type = reflect.TypeOf((*error)(nil)).Elem()

// ErrorType returns the reflection type for the error interface
func ErrorType() reflect.Type {
	return errorType
}

// inType is the cached reflection lookup for fx.In
var inType reflect.Type = reflect.TypeOf(fx.In{})

// InType returns the reflection type of fx.In
func InType() reflect.Type {
	return inType
}

// outType is the cached reflection lookup for fx.Out
var outType reflect.Type = reflect.TypeOf(fx.Out{})

// OutType returns the reflection type of fx.Out
func OutType() reflect.Type {
	return outType
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

// ValueOf is a convenient utility function for turning v into a reflect.Value.
// If v is already a reflect.Value, it is returned as is.  Otherwise, the result
// of reflect.ValueOf(v) is returned.
func ValueOf(v interface{}) reflect.Value {
	if vv, ok := v.(reflect.Value); ok {
		return vv
	}

	return reflect.ValueOf(v)
}

// TypeOf is a convenient utility function for turning a v into a reflect.Type.
// If v is already a reflect.Type, it is returned as is.  If v is a reflect.Value,
// v.Type() is returned.  Otherwise, the result of reflect.TypeOf(v) is returned.
func TypeOf(v interface{}) reflect.Type {
	if vv, ok := v.(reflect.Value); ok {
		return vv.Type()
	} else if vt, ok := v.(reflect.Type); ok {
		return vt
	}

	return reflect.TypeOf(v)
}

// Target describes a sink for an unmarshal operation.
//
// Viper requires a pointer to be passed to its UnmarshalXXX functions.  However,
// this package uses a prototype pattern whereby a caller may specify a pointer
// or a value.  A target bridges that gap by storing the results of reflection
// from NewTarget to give a consistent way of referring to the actual object
// that should be unmarshaled as opposed to the object produced for dependency injection.
type Target struct {
	// Component refers the the actual value that should be returned from an uber/fx constructor.
	// This holds the value of the actual component that participates in dependency injection.
	Component reflect.Value

	// UnmarshalTo is the value that should be unmarshaled.  This value is always a pointer.
	// If Component is a pointer, UnmarshalTo will be the same value.  Otherwise, UnmarshalTo
	// will be a pointer the the Component value.
	UnmarshalTo reflect.Value
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
//	NewTarget(Config{})
//	NewTarget(Config{Timeout: 15 * time.Second}) // a default value for Timeout
//
// can be used with:
//
//	fx.New(
//	  fx.Invoke(
//	    func(cfg Config) {},
//	  ),
//	)
//
// (2) The prototype may be a non-nil pointer to a struct.  A new struct will be
// allocated with fields set to the same values as the prototype prior to
// unmarshaling.  The component will be pointer to this new struct.
//
//	NewTarget(&Config{})
//	NewTarget(new(Config))
//	NewTarget(&Config{Timeout: 15 * time.Second}) // a default value for Timeout
//
// can be used with:
//
//	fx.New(
//	  fx.Invoke(
//	    func(cfg *Config) {
//	      // always a non-nil pointer, but any fields not unmarshaled
//	      // will be set to their corresponding fields in the prototype
//	    },
//	  ),
//	)
//
// (3) The prototype may be a nil pointer to a struct.  A new struct of the same type
// will be created, but with all fields set to their zero values prior to unmarshaling.
// The component will be a pointer to this new struct.
//
//	NewTarget((*Config)(nil))
//
// can be used with:
//
//	fx.New(
//	  fx.Invoke(
//	    func(cfg *Config) {
//	      // always a non-nil pointer, but any fields not unmarshaled
//	      // will be set to their zero values
//	    },
//	  ),
//	)
//
// If the prototype does not refer to a struct, the results of this function are undefined.
func NewTarget(prototype interface{}) (t Target) {
	pvalue := ValueOf(prototype)
	if pvalue.Kind() == reflect.Ptr {
		t.UnmarshalTo = reflect.New(pvalue.Type().Elem())
		if !pvalue.IsNil() {
			t.UnmarshalTo.Elem().Set(pvalue.Elem())
		}

		t.Component = t.UnmarshalTo
	} else {
		t.UnmarshalTo = reflect.New(pvalue.Type())
		t.UnmarshalTo.Elem().Set(pvalue)
		t.Component = t.UnmarshalTo.Elem()
	}

	return
}

// TryConvert provides a more flexible alternative to a switch/type block.  It reflects
// the src parameter using ValueOf in this package, then determines which of a set of case
// functions to invoke based on the sole input parameter of each callback.  Exactly zero or one
// case function is invoked.  This function returns true if a callback was invoked, which
// means a conversion was successful.  Otherwise, this function returns false to indicate
// that no conversion to the available callbacks was possible.
//
// The src parameter may be a regular value or a reflect.Value.  It may refer to a scalar value,
// an array, or a slice.
//
// Each case is checked for a match first by a simple direct conversion.  If that is unsuccessful,
// then if both the src and the case refer to sequences, an attempt is made to convert each element
// into a slice that matches the case.  Failing both of those attempts, the next cases is considered.
//
// In many dependency injection situations, looser type conversions than what golang allows
// are preferable.  For example, gorilla/mux.MiddlewareFunc and justinas/alice.Constructor
// are not considered the same types by golang, even though they are both func(http.Handler) http.Handler.
// Using TryConvert allows arrange to support multiple middleware packages without actually
// having to import those packages just for the types.
func TryConvert(src interface{}, cases ...interface{}) bool {
	var (
		from         = ValueOf(src)
		fromSequence = (from.Kind() == reflect.Array || from.Kind() == reflect.Slice)
	)

	for _, c := range cases {
		var (
			cf = reflect.ValueOf(c)
			to = cf.Type().In(0)
		)

		// first, try a direct conversion
		if from.Type().ConvertibleTo(to) {
			cf.Call([]reflect.Value{
				from.Convert(to),
			})

			return true
		}

		// next, try to convert elements of one sequence into another
		// NOTE: we don't support converting to arrays, only slices
		if fromSequence && to.Kind() == reflect.Slice {
			if from.Type().Elem().ConvertibleTo(to.Elem()) {
				s := reflect.MakeSlice(
					to,         // to is a slice type already
					from.Len(), // len
					from.Len(), // cap
				)

				for i := 0; i < from.Len(); i++ {
					s.Index(i).Set(
						from.Index(i).Convert(to.Elem()),
					)
				}

				cf.Call([]reflect.Value{s})
				return true
			}
		}
	}

	return false
}
