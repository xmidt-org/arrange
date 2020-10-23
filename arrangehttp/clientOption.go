package arrangehttp

import (
	"net/http"

	"github.com/xmidt-org/arrange"
)

// ClientOption is a functional option type that configures an http.Client.
type ClientOption func(*http.Client) error

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

// newClientOption reflects v to produce a ClientOption.  If v cannot be converted
// to a ClientOption, this function returns nil
func newClientOption(v interface{}) ClientOption {
	var co ClientOption
	arrange.TryConvert(
		v,
		func(v ClientOption) {
			co = v
		},
		func(chain ClientMiddlewareChain) {
			co = func(client *http.Client) error {
				client.Transport = chain.Then(client.Transport)
				return nil
			}
		},
		func(ctor RoundTripperConstructor) {
			co = func(client *http.Client) error {
				client.Transport = NewRoundTripperChain(ctor).Then(client.Transport)
				return nil
			}
		},
		// support value groups
		func(ctors []RoundTripperConstructor) {
			co = func(client *http.Client) error {
				client.Transport = NewRoundTripperChain(ctors...).Then(client.Transport)
				return nil
			}
		},
	)

	return co
}
