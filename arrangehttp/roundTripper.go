package arrangehttp

import (
	"net/http"
	"reflect"

	"github.com/xmidt-org/arrange"
)

// RoundTripperFunc is a function type that implements http.RoundTripper.
// Useful for simple decoration and testing.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper
func (rtf RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rtf(r)
}

// RoundTripperConstructor is a strategy for decorating an http.RoundTripper.
// Typical use cases are metrics and logging.
type RoundTripperConstructor func(http.RoundTripper) http.RoundTripper

// RoundTripperChain is a sequence of RoundTripperConstructors.  A RoundTripperChain is immutable,
// and will apply its constructors in order.  The zero value for this type is a valid,
// empty chain that will not decorate anything.
type RoundTripperChain struct {
	c []RoundTripperConstructor
}

// NewRoundTripperChain creates a chain from a sequence of constructors.  The constructors
// are always applied in the order presented here.
func NewRoundTripperChain(c ...RoundTripperConstructor) RoundTripperChain {
	return RoundTripperChain{
		c: append([]RoundTripperConstructor{}, c...),
	}
}

// Append adds additional RoundTripperConstructors to this chain, and returns the new chain.
// This chain is not modified.  If more has zero length, this chain is returned.
func (lc RoundTripperChain) Append(more ...RoundTripperConstructor) RoundTripperChain {
	if len(more) > 0 {
		return RoundTripperChain{
			c: append(
				append([]RoundTripperConstructor{}, lc.c...),
				more...,
			),
		}
	}

	return lc
}

// Extend is like Append, except that the additional RoundTripperConstructors come from
// another chain
func (lc RoundTripperChain) Extend(more RoundTripperChain) RoundTripperChain {
	return lc.Append(more.c...)
}

// Then decorates the given Listen strategy with all of the constructors
// applied, in the order they were presented to this chain.  If next is
// nil, then the returned RoundTripper will decorate http.DefaultTransport.
// If this chain is empty, this method simply returns next, even if next is nil.
func (lc RoundTripperChain) Then(next http.RoundTripper) http.RoundTripper {
	if len(lc.c) > 0 {
		if next == nil {
			next = http.DefaultTransport
		}

		// apply in reverse order, so that the order of
		// execution matches the order supplied to this chain
		for i := len(lc.c) - 1; i >= 0; i-- {
			next = lc.c[i](next)
		}
	}

	return next
}

type ClientMiddlewareChain interface {
	Then(http.RoundTripper) http.RoundTripper
}

var roundTripperConstructorType = reflect.TypeOf(
	func(http.RoundTripper) http.RoundTripper { return nil },
)

// TryClientMiddlewareChain attempts to convert v into a ClientMiddlewareChain.  If
// the conversion is unsuccessful, this function returns nil.
//
// If v implements ClientMiddlewareChain, it is returned as is.
//
// If v is convertible to RoundTripperConstructor, it is return wrapped
// in a RoundTripperChain.
//
// If v is an array or slice of an element convertible to RoundTripperConstructor,
// each element is converted appropriately and returned wrapped in a RoundTripperChain.
func TryClientMiddlewareChain(v interface{}) ClientMiddlewareChain {
	if chain, ok := v.(ClientMiddlewareChain); ok {
		return chain
	}

	vv := arrange.ValueOf(v)
	switch {
	case vv.Kind() == reflect.Array:
		fallthrough

	case vv.Kind() == reflect.Slice:
		if vv.Type().Elem().ConvertibleTo(roundTripperConstructorType) {
			dst := reflect.MakeSlice(
				roundTripperConstructorType, // element type
				vv.Len(),                    // len
				vv.Len(),                    // cap
			)

			for i := 0; i < vv.Len(); i++ {
				dst.Index(i).Set(
					vv.Index(i).Convert(roundTripperConstructorType),
				)
			}

			return NewRoundTripperChain(
				vv.Convert(roundTripperConstructorType).Interface().([]RoundTripperConstructor)...,
			)
		}

	case vv.Type().ConvertibleTo(roundTripperConstructorType):
		return NewRoundTripperChain(
			vv.Convert(roundTripperConstructorType).Interface().(RoundTripperConstructor),
		)
	}

	return nil
}
