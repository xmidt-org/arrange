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

// NewErrorValue is a convenience for safely producing a reflect.Value from an error.
// Useful when creating function stubs for reflect.MakeFunc.
func NewErrorValue(err error) reflect.Value {
	errPtr := reflect.New(ErrorType())
	if err != nil {
		errPtr.Elem().Set(reflect.ValueOf(err))
	}

	return errPtr.Elem()
}

// Target describes a sink for an unmarshal operation.
//
// Viper requires a pointer to be passed to its UnmarshalXXX functions.  However,
// this package uses a prototype pattern whereby a caller may specify a pointer
// or a value.  A target bridges that gap by storing the results of reflection
// from NewTarget to give a consistent way of referring to the actual object
// that should be unmarshaled as opposed to the object produced for dependency injection.
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

// VisitResult is the enumerated constant returned by a FieldVisitor
type VisitResult int

const (
	// VisitContinue indicates that field visitation should continue as normal
	VisitContinue VisitResult = iota

	// VisitSkip indicates that the fields of an embedded struct should not be visited.
	// If returned for any other kind of field, this is equivalent to VisitContinue.
	VisitSkip

	// VisitTerminate terminates the tree walk immediately
	VisitTerminate
)

// FieldVisitor is a strategy for visiting each exported field of a struct
type FieldVisitor func(reflect.StructField, reflect.Value) VisitResult

// VisitFields walks the tree of struct fields.  Each embedded struct is also
// traversed, but named struct fields are not.  Unexported fields are never traversed.
//
// If root is actually a reflect.Value, that value will be used or dereferenced if
// it is a pointer.
//
// If root is a struct or any level of pointer to a struct, it will be dereferenced
// and used as the starting point.
//
// If root is not a struct, or cannot be dereferenced to a struct, this function
// returns an invalid value, i.e. IsValid() will return false.  Also, an invalid
// value is returned if root is a nil pointer.
//
// If any traversal occurred, this function returns the actual reflect.Value representing
// the struct that was the root of the tree traversal.
func VisitFields(root interface{}, v FieldVisitor) reflect.Value {
	var rv reflect.Value
	if rt, ok := root.(reflect.Value); ok {
		rv = rt
	} else {
		rv = reflect.ValueOf(root)
	}

	// dereference as much as needed
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			// can't traverse into a nil
			return rv
		}

		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return reflect.ValueOf(nil)
	}

	stack := []reflect.Value{rv}
	for len(stack) > 0 {
		var (
			end = len(stack) - 1
			s   = stack[end]
			st  = s.Type()
		)

		stack = stack[:end]
		for i := 0; i < st.NumField(); i++ {
			f := st.Field(i)
			if len(f.PkgPath) > 0 {
				// NOTE: don't consider unexported fields
				continue
			}

			fv := s.Field(i)
			if r := v(f, fv); r == VisitTerminate {
				return rv
			} else if f.Anonymous && r != VisitSkip {
				stack = append(stack, fv)
			}
		}
	}

	return rv
}

// IsIn performs a struct field traversal to find an fx.In embedded
// struct.  If v could not be traversed or does not embed fx.In, this
// function returns an invalid reflect.Value and false.  Otherwise,
// the reflect.Value representing the actual struct, possibly dereferenced,
// is returned along with true.
func IsIn(v interface{}) (reflect.Value, bool) {
	result := VisitContinue
	root := VisitFields(
		v,
		func(f reflect.StructField, fv reflect.Value) VisitResult {
			if f.Anonymous && f.Type == InType() {
				result = VisitTerminate
			}

			return result
		},
	)

	return root, root.IsValid() && result == VisitTerminate
}
