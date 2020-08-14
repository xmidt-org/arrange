package arrangehttp

import "net/http"

// emptyHeader is an internal singleton representing a blank Header
var emptyHeader = Header{}

// Header is an immutable set of HTTP headers useful for keeping
// deep copies of http.Header objects, such as when decorating
// http.Handlers.
//
// The zero value of this type is an immutable, empty Header that
// can be used as a sentinel value.
//
// All header keys are stored internally as canonicalized values.  All header
// values will be deep copies of any values used in initialization.
type Header struct {
	h http.Header
}

// NewHeader makes a deep copy of the given source with each
// key filtered through http.CanonicalNameKey.  This is useful when
// storing an http.Header longterm, such as when decorating an http.Handler.
//
// If src is empty or nil, an empty Header is returned.  Otherwise, the returned
// Header is a distinct, deep copy of the source.
func NewHeader(src http.Header) Header {
	if len(src) > 0 {
		cleaned := make(http.Header, len(src))
		for key, values := range src {
			if len(key) > 0 && len(values) > 0 {
				key = http.CanonicalHeaderKey(key)
				cleaned[key] = append([]string{}, values...)
			}
		}

		if len(cleaned) > 0 {
			return Header{h: cleaned}
		}
	}

	return emptyHeader
}

// NewHeaderFromMap is a simpler version of NewHeader, allowing one
// to specify a plain map of strings.  In all other ways this function
// is identifical to NewHeader.
func NewHeaderFromMap(src map[string]string) Header {
	if len(src) > 0 {
		cleaned := make(http.Header, len(src))
		for key, value := range src {
			if len(key) > 0 {
				key = http.CanonicalHeaderKey(key)
				cleaned[key] = []string{value}
			}
		}

		if len(cleaned) > 0 {
			return Header{h: cleaned}
		}
	}

	return emptyHeader
}

// NewHeaders provides a variadic way of constructing an immutable Header.
// The sequence of strings is expected to be in key/value pair order.  Duplicate
// keys are allowed.  If the number of values is odd, the last value is a
// header key with an empty value.
//
// In all other ways this function is identifical to NewHeader.
func NewHeaders(src ...string) Header {
	cleaned := make(http.Header, len(src)/2)
	for i, j := 0, 1; i < len(src); i, j = i+2, j+2 {
		if len(src[i]) > 0 {
			key := http.CanonicalHeaderKey(src[i])
			if j < len(src) {
				cleaned[key] = append(cleaned[key], src[j])
			} else {
				// dangling key!
				cleaned[key] = []string{""}
			}
		}
	}

	if len(cleaned) > 0 {
		return Header{h: cleaned}
	}

	return emptyHeader
}

// Len returns the count of keys in this header
func (h Header) Len() int {
	return len(h.h)
}

// AddTo appends this Header's key/values to the given http.Header.
// Because a Header already contains canonicalized header keys, this
// method is more efficient than http.Header.Add.
func (h Header) AddTo(dst http.Header) {
	for key, values := range h.h {
		dst[key] = append(dst[key], values...)
	}
}

// AddResponse is a middleware that simply adds all headers to
// the response.  If this Header is empty, no decoration is performed.
func (h Header) AddResponse(next http.Handler) http.Handler {
	if h.Len() == 0 {
		return next
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		h.AddTo(response.Header())
		next.ServeHTTP(response, request)
	})
}

// AddRequest is a RoundTripperConstructor that adds all headers to
// the request.  If this Header is empty, no decoration is performed.
func (h Header) AddRequest(next http.RoundTripper) http.RoundTripper {
	if h.Len() == 0 {
		return next
	}

	return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header == nil {
			request.Header = make(http.Header, h.Len())
		}

		h.AddTo(request.Header)
		return next.RoundTrip(request)
	})
}
