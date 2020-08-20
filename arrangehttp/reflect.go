package arrangehttp

import "reflect"

// tryConvertToOptionSlice takes a reflect.Value and tries to convert into a slice
// of an option type supported by the S builder.  For example, []SOption, []mux.MiddlewareFunc, etc.
// Scalars that are supported are converto a slice of length 1.  This function
// always returns a slice whose element type is the same as optionType.
//
// If the second return value is true, the interface{} will be castable to a slice
// of the optionType parameter, e.g. tryConvertToOptionSlice(v, SOption(nil)) would
// always return []SOption if successful.
//
// This function is used by both NewSOption and NewCOption to discover supported options.
func tryConvertToOptionSlice(v reflect.Value, optionType interface{}) (interface{}, bool) {
	ot := reflect.TypeOf(optionType)
	switch {
	case v.Kind() == reflect.Array:
		// not sure anyone would use an actual array, but it's trivial to support
		fallthrough

	case v.Kind() == reflect.Slice:
		if v.Type().Elem().ConvertibleTo(ot) {
			s := reflect.MakeSlice(
				reflect.SliceOf(ot), // element type
				v.Len(),             // len
				v.Len(),             // cap
			)

			for i := 0; i < v.Len(); i++ {
				s.Index(i).Set(
					v.Index(i).Convert(ot),
				)
			}

			return s.Interface(), true
		}

	case v.Type().ConvertibleTo(ot):
		s := reflect.MakeSlice(
			reflect.SliceOf(ot), // element type
			1,                   // len
			1,                   // cap
		)

		s.Index(0).Set(
			v.Convert(ot),
		)

		return s.Interface(), true
	}

	return nil, false
}
