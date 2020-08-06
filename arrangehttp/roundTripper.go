package arrangehttp

import "net/http"

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
