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

// IsIn tests if the given value refers to a struct that embeds fx.In.
// Embedded, exported fields are searched recursively.  If t does not
// refer to a struct, this function returns false.  The struct must embed
// fx.In, not simply have fx.In as a field.
//
// IsIn returns both the actual reflect.Type that it inspected together
// with whether that type is a struct that embeds fx.In.
func IsIn(v interface{}) (reflect.Type, bool) {
	t := TypeOf(v)
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if !f.Anonymous {
				// skip
				continue
			}

			if f.Type == InType() {
				return t, true
			}

			if len(f.PkgPath) == 0 {
				// only recurse for anonymous (embedded), exported fields
				if _, ok := IsIn(f.Type); ok {
					return t, true
				}
			}
		}
	}

	return t, false
}

// FieldVisitor is a strategy for visiting each exported field of a struct
type FieldVisitor func(reflect.StructField, reflect.Value) bool

// IsOptional tests if the given struct field is tagged as an optional field.
// Only applicable for structs that embed fx.In.
//
// Since there is no way to tell if an fx.App actually set a field when it is
// optional, this function in tandem with checking for a zero value is a way
// to ignore fields for components that were not supplied.
func IsOptional(f reflect.StructField) bool {
	return f.Tag.Get("optional") == "true"
}

// VisitDependencies walks the tree of struct fields looking for things that are acceptable
// as injected dependencies, whether or not the given struct embeds fx.In.  This method ensures
// that only exported struct fields for which IsValid() returns true are visited.  Anonymous
// fields that are exported will be visited and, if they are structs, will be recursively visited.
//
// If root is actually a reflect.Value, that value will be used.  Otherwise, it's assumed that
// root is a struct.  This function will dereference pointers to any depth.
//
// If root is not a struct, or cannot be dereferenced to a struct, this function does nothing.
//
// Visitation continues until there are no more fields or until v returns false.
func VisitDependencies(root interface{}, v FieldVisitor) {
	rv := ValueOf(root)

	// dereference as much as necessary
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return
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
			var (
				f  = st.Field(i)
				fv = s.Field(i)
			)

			if len(f.PkgPath) > 0 || !fv.IsValid() || !fv.CanInterface() {
				// NOTE: skip unexported fields or those whose value cannot be accessed
				continue
			}

			if !v(f, fv) {
				return
			}

			if f.Anonymous {
				// NOTE: any anonymous, exported field will be recursively visited
				stack = append(stack, fv)
			}
		}
	}
}

// TryConvert attempts to convert dst into a slice of whatever type src is.  If src is
// itself a slice, then an attempt each made to convert each element of src into the dst type.
//
// The ConvertibleTo method in the reflect package is used to determine if conversion
// is possible.  If it is, then this function always returns a slice of the type
// referred to by dst.  This simplifies consumption of the result, as a caller may
// always safely cast it to a "[]dst" if the second return value is true.
//
// The src parameter may be an actual object or a reflect.Value.  The src may also be a slice
// type instead of a scalar.
//
// The dst parameter may be an actual object, a reflect.Value, or a reflect.Type.
//
// This function is useful in dependency injection situations when the
// allowed type should be looser than what golang allows.  For example, allowing
// a "func(http.Handler) http.Handler" where a "gorilla/mux.MiddlewareFunc" is desired.
//
// This function returns a nil interface{} and false if the conversion was not possible.
func TryConvert(dst, src interface{}) (interface{}, bool) {
	var (
		from = ValueOf(src)
		to   = TypeOf(dst)
	)

	switch {
	case from.Kind() == reflect.Array:
		fallthrough

	case from.Kind() == reflect.Slice:
		if from.Type().Elem().ConvertibleTo(to) {
			s := reflect.MakeSlice(
				reflect.SliceOf(to), // element type
				from.Len(),          // len
				from.Len(),          // cap
			)

			for i := 0; i < from.Len(); i++ {
				s.Index(i).Set(
					from.Index(i).Convert(to),
				)
			}

			return s.Interface(), true
		}

	case from.Type().ConvertibleTo(to):
		s := reflect.MakeSlice(
			reflect.SliceOf(to), // element type
			1,                   // len
			1,                   // cap
		)

		s.Index(0).Set(
			from.Convert(to),
		)

		return s.Interface(), true
	}

	return nil, false
}
