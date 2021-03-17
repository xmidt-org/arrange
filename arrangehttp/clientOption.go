package arrangehttp

import (
	"net/http"

	"github.com/xmidt-org/arrange"
)

// clientInfo is the bucket of information about a client under construction.
// Functional options modify this type.
type clientInfo struct {
	client     *http.Client
	middleware RoundTripperChain
}

// applyMiddleware decorates the http.RoundTripper associated with the client.
// If the client.Target field is set, that field is decorated.  Otherwise,
// http.DefaultTarget is decorated.
func (ci *clientInfo) applyMiddleware() {
	next := ci.client.Transport
	if next == nil {
		next = http.DefaultTransport
	}

	ci.client.Transport = ci.middleware.Then(next)
}

// cOption is the internal functional option used to tailor http.Clients
type cOption func(*clientInfo) error

// newCOption reflects v to produce a cOption.  If v cannot be converted
// to a ClientOption, this function returns nil
func newCOption(v interface{}) cOption {
	var co cOption
	arrange.TryConvert(
		v,
		func(v ClientOption) {
			co = v.cOption
		},
		func(chain ClientMiddlewareChain) {
			co = func(ci *clientInfo) error {
				ci.middleware = ci.middleware.Append(chain.Then)
				return nil
			}
		},
		func(ctor RoundTripperConstructor) {
			co = func(ci *clientInfo) error {
				ci.middleware = ci.middleware.Append(ctor)
				return nil
			}
		},
		// support value groups
		func(ctors []RoundTripperConstructor) {
			co = func(ci *clientInfo) error {
				ci.middleware = ci.middleware.Append(ctors...)
				return nil
			}
		},
	)

	return co
}

// ClientOption is a functional option type that configures an http.Client.
type ClientOption func(*http.Client) error

// cOption essentially converts this ClientOption into the internal option type
func (co ClientOption) cOption(ci *clientInfo) error {
	return co(ci.client)
}

// ClientOptions glues together multiple options into a single, immutable option.
// Use this to produce aggregate options within an fx.App, instead of an []ClientOption.
func ClientOptions(o ...ClientOption) ClientOption {
	if len(o) == 1 {
		return o[0]
	}

	return func(client *http.Client) error {
		for _, f := range o {
			if err := f(client); err != nil {
				return err
			}
		}

		return nil
	}
}
