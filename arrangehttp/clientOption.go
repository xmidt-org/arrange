package arrangehttp

import (
	"net/http"
	"reflect"
	"strings"

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
	// Apply modifies the given client.
	Apply(*http.Client) error
}

// ClientOptionFunc is a function type that implements ClientOption.
type ClientOptionFunc func(*http.Client) error

func (cof ClientOptionFunc) Apply(c *http.Client) error {
	return cof(c)
}

// ClientOptions is an aggregate set of ClientOption that acts as a single option.
type ClientOptions []ClientOption

// Apply invokes each option in order.  Options are always invoked, even when
// one or more errors occur.  The returned error may be an aggregate error
// and can always be inspected via go.uber.org/multierr.
func (co ClientOptions) Apply(c *http.Client) (err error) {
	for _, o := range co {
		err = multierr.Append(err, o.Apply(c))
	}

	return
}

// Add appends options to this slice.  Each value is converted to a ClientOption
// via AsClientOption.
func (co *ClientOptions) Add(opts ...any) {
	if len(opts) == 0 {
		return
	}

	if cap(*co) < (len(*co) + len(opts)) {
		bigger := make(ClientOptions, 0, len(*co)+len(opts))
		bigger = append(bigger, *co...)
		*co = bigger
	}

	for _, o := range opts {
		*co = append(*co, AsClientOption(o))
	}
}

// AsClientOption converts a value into a ClientOption.  This function never returns nil
// and does not panic if v cannot be converted.
//
// Any of the following kinds of values can be converted:
//   - any type that implements ClientOption
//   - any type that supplies an Apply(*http.Client) method that returns no error
//   - an underlying type of func(*http.Client)
//   - an underlying type of func(*http.Client) error
//
// Any other kind of value will result in a ClientOption that returns an error indicating
// that the type cannot be converted.
func AsClientOption(v any) ClientOption {
	type clientOptionNoError interface {
		Apply(*http.Client)
	}

	if co, ok := v.(ClientOption); ok {
		return co
	} else if co, ok := v.(clientOptionNoError); ok {
		return ClientOptionFunc(func(c *http.Client) error {
			co.Apply(c)
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
