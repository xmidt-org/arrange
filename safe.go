package arrange

import "reflect"

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
func Safe[T any](candidate, def T) T {
	if cv := reflect.ValueOf(candidate); !cv.IsValid() || (cv.Kind() == reflect.Ptr && cv.IsNil()) {
		return def
	}

	return candidate
}
