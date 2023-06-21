package arrangereflect

import (
	"reflect"
)

// Safe returns a safe instance of T.  The candidate is used if it is
// a valid, non-nil instance.  Otherwise, the def value is used.
//
// A primary motivation for this function is decoration, e.g. server middleware:
//
//	var hf http.HandlerFunc // uninitialized
//	func use(v http.Handler) {
//	    if v != nil {
//		    v.ServeHTTP(...)
//	    }
//	}
//
//	use(hf) // this will panic
//	use(arrange.Safe[http.Handler](hf, http.DefaultServeMux)) // this will be fine
func Safe[T any](candidate, def T) (result T) {
	result = def
	defer func() {
		// allow IsNil to panic instead of trying all possible types
		if r := recover(); r != nil {
			// IsNil panicked, which means candidate wasn't a type that could be nil
			result = candidate
		}
	}()

	if cv := reflect.ValueOf(candidate); cv.IsValid() && !cv.IsNil() {
		result = candidate
	}

	return
}
