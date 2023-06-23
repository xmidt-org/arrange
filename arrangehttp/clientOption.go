package arrangehttp

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/xmidt-org/arrange/internal/arrangereflect"
	"go.uber.org/multierr"
)

// InvalidClientOptionTypeError is returned by a ClientOption produced by AsClientOption
// to indicate that a type could not be converted.
type InvalidClientOptionTypeError struct {
	Type reflect.Type
}

// Error describes the type that could not be converted.
func (icote *InvalidClientOptionTypeError) Error() string {
	var o strings.Builder
	o.WriteString(icote.Type.String())
	o.WriteString(" cannot be converted to a ClientOption")

	return o.String()
}

// ClientOption is a general-purpose modifier for an *http.Client.  Typically, these
// will be created as value group within an enclosing *fx.App.
type ClientOption interface {
	// ApplyToClient modifies the given client.  This method can return an error to
	// indicate that the option was incorrectly applied.
	ApplyToClient(*http.Client) error
}

// ClientOptionFunc is a function type that implements ClientOption.
type ClientOptionFunc func(*http.Client) error

func (cof ClientOptionFunc) ApplyToClient(c *http.Client) error {
	return cof(c)
}

// ClientOptions is an aggregate set of ClientOption that acts as a single option.
type ClientOptions []ClientOption

// ApplyToClient invokes each option in order.  Options are always invoked, even when
// one or more errors occur.  The returned error may be an aggregate error
// and can always be inspected via go.uber.org/multierr.
func (co ClientOptions) ApplyToClient(c *http.Client) (err error) {
	for _, o := range co {
		err = multierr.Append(err, o.ApplyToClient(c))
	}

	return
}

// AsClientOption converts a value into a ClientOption.  This function never returns nil
// and does not panic if v cannot be converted.
//
// Any of the following kinds of values can be converted:
//   - any type that implements ClientOption
//   - any type that supplies an ApplyToClient(*http.Client) method that returns no error
//   - an underlying type of func(*http.Client)
//   - an underlying type of func(*http.Client) error
//
// Any other kind of value will result in a ClientOption that returns an error indicating
// that the type cannot be converted.
func AsClientOption(v any) ClientOption {
	type clientOptionNoError interface {
		ApplyToClient(*http.Client)
	}

	if co, ok := v.(ClientOption); ok {
		return co
	} else if co, ok := v.(clientOptionNoError); ok {
		return ClientOptionFunc(func(c *http.Client) error {
			co.ApplyToClient(c)
			return nil
		})
	} else if f, ok := v.(func(*http.Client) error); ok {
		return ClientOptionFunc(f)
	} else if f, ok := v.(func(*http.Client)); ok {
		return ClientOptionFunc(func(c *http.Client) error {
			f(c)
			return nil
		})
	}

	return ClientOptionFunc(func(_ *http.Client) error {
		return &InvalidClientOptionTypeError{
			Type: reflect.TypeOf(v),
		}
	})
}

// ClientMiddlewareFunc is the underlying type for round tripper decoration.
type ClientMiddlewareFunc interface {
	~func(http.RoundTripper) http.RoundTripper
}

// ClientMiddleware returns a ClientOption that applies the given middleware
// to the Transport (http.RoundTripper).
func ClientMiddleware[M ClientMiddlewareFunc](fns ...M) ClientOption {
	return AsClientOption(func(c *http.Client) {
		c.Transport = arrangereflect.Decorate(
			arrangereflect.Safe[http.RoundTripper](c.Transport, http.DefaultTransport),
			fns...,
		)
	})
}
