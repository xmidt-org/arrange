package arrangehttp

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// ClientFactory is the interface implemented by unmarshaled configuration objects
// that produces an http.Client.  The default implementation of this interface is ClientConfig.
type ClientFactory interface {
	NewClient() (*http.Client, error)
}

// TransportConfig holds the unmarshalable configuration options for building an http.Transport
type TransportConfig struct {
	TLSHandshakeTimeout    time.Duration
	DisableKeepAlives      bool
	DisableCompression     bool
	MaxIdleConns           int
	MaxIdleConnsPerHost    int
	MaxConnsPerHost        int
	IdleConnTimeout        time.Duration
	ResponseHeaderTimeout  time.Duration
	ExpectContinueTimeout  time.Duration
	ProxyConnectHeader     http.Header
	MaxResponseHeaderBytes int64
	WriteBufferSize        int
	ReadBufferSize         int
	ForceAttemptHTTP2      bool
}

// NewTransport creates an http.Transport using this unmarshaled configuration
// together with TLS information
func (tc TransportConfig) NewTransport(c *arrangetls.Config) (transport *http.Transport, err error) {
	transport = &http.Transport{
		TLSHandshakeTimeout:    tc.TLSHandshakeTimeout,
		DisableKeepAlives:      tc.DisableKeepAlives,
		DisableCompression:     tc.DisableCompression,
		MaxIdleConns:           tc.MaxIdleConns,
		MaxIdleConnsPerHost:    tc.MaxIdleConnsPerHost,
		MaxConnsPerHost:        tc.MaxConnsPerHost,
		IdleConnTimeout:        tc.IdleConnTimeout,
		ResponseHeaderTimeout:  tc.ResponseHeaderTimeout,
		ExpectContinueTimeout:  tc.ExpectContinueTimeout,
		ProxyConnectHeader:     tc.ProxyConnectHeader,
		MaxResponseHeaderBytes: tc.MaxResponseHeaderBytes,
		WriteBufferSize:        tc.WriteBufferSize,
		ReadBufferSize:         tc.ReadBufferSize,
		ForceAttemptHTTP2:      tc.ForceAttemptHTTP2,
	}

	transport.TLSClientConfig, err = c.New()
	return
}

// ClientConfig holds unmarshaled client configuration options.  It is the
// built-in ClientFactory implementation in this package.
type ClientConfig struct {
	Timeout   time.Duration
	Transport TransportConfig
	TLS       *arrangetls.Config
}

// NewClient produces an http.Client given these unmarshaled configuration options
func (cc ClientConfig) NewClient() (client *http.Client, err error) {
	client = &http.Client{
		Timeout: cc.Timeout,
	}

	client.Transport, err = cc.Transport.NewTransport(cc.TLS)
	return
}

// COption is a functional option for a fluent client builder
type COption func(*http.Client) error

// COptions aggregates several COption instances into a single option
func COptions(options ...COption) COption {
	if len(options) == 1 {
		return options[0]
	}

	return func(c *http.Client) error {
		var err error
		for _, co := range options {
			err = co(c)
			if err != nil {
				break
			}
		}

		return err
	}
}

// NewCOption reflects an object and tries to convert it into an COption.  The set
// of types allowed is flexible:
//
//   (1) COption or any type convertible to a COption
//   (2) Any type convertible to func(*http.Client), which is basically a COption that returns no error
//   (3) RoundTripperConstructor are any type convertible to RoundTripperConstructor
//   (4) RoundTripperChain
//   (5) A slice or array of any of the above, which are applied in slice element order
//
// Any other type will produce an error.
func NewCOption(o interface{}) (COption, error) {
	v := reflect.ValueOf(o)

	// handled types noted below:

	// COption
	// []COption
	if o, ok := tryConvertToOptionSlice(v, COption(nil)); ok {
		return COptions(o.([]COption)...), nil
	}

	// explicitly support a COption variant that returns no error
	// this helps reduce code noise when there are lots of options,
	// avoiding "return nil" all over the place
	if o, ok := tryConvertToOptionSlice(v, (func(*http.Client))(nil)); ok {
		return func(c *http.Client) error {
			for _, f := range o.([]func(*http.Client)) {
				f(c)
			}

			return nil
		}, nil
	}

	// RoundTripperConstructor
	// []RoundTripperConstructor
	// func(http.RoundTripper) http.RoundTripper
	// []func(http.RoundTripper) http.RoundTripper
	if rc, ok := tryConvertToOptionSlice(v, RoundTripperConstructor(nil)); ok {
		return func(c *http.Client) error {
			c.Transport = NewRoundTripperChain(rc.([]RoundTripperConstructor)...).Then(c.Transport)
			return nil
		}, nil
	}

	// RoundTripperChain
	// []RoundTripperChain
	if rtc, ok := tryConvertToOptionSlice(v, RoundTripperChain{}); ok {
		return func(c *http.Client) error {
			for _, chain := range rtc.([]RoundTripperChain) {
				c.Transport = chain.Then(c.Transport)
			}

			return nil
		}, nil
	}

	return nil, fmt.Errorf("%s is not supported as a COption", reflect.TypeOf(o))
}

// ClientIn is the set of dependencies required to build an *http.Client component
type ClientIn struct {
	// Unmarshaler is the required arrange Unmarshaler component used to unmarshal
	// a ClientFactory
	Unmarshaler arrange.Unmarshaler

	// Printer is the optional fx.Printer used to output informational messages about
	// client unmarshaling and configuration.  If unset, arrange.DefaultPrinter() is used.
	Printer fx.Printer `optional:"true"`

	// Lifecycle is used to bind http.Client.CloseIdleConnections to the
	// fx.App OnStop event
	Lifecycle fx.Lifecycle
}

// C is a Fluent Builder for creating an http.Client as an uber/fx component.
// This type should be constructed with the Client function.
type C struct {
	errs         []error
	options      []COption
	dependencies []reflect.Type
	prototype    ClientFactory
}

// Client begins a Fluent Builder chain for constructing an http.Client from
// unmarshaled configuration and introducing that http.Client as a component
// for an enclosing uber/fx app.
func Client(o ...interface{}) *C {
	return new(C).
		ClientFactory(ClientConfig{}).
		Use(o...)
}

// ClientFactory sets the prototype factory that is unmarshaled from Viper.
// This prototype obeys the rules of arrange.NewTarget.  By default, ClientConfig
// is used as the ClientFactory.  This build method allows a caller to use
// custom configuration.
func (c *C) ClientFactory(prototype ClientFactory) *C {
	c.prototype = prototype
	return c
}

// Use applies options to this builder.  The set of types allowed are any
// of the types that can be supplied to NewCOption as well as instances
// of structs embedded with fx.In.
//
// Anything convertible to an COption is evaluated at construction time.
//
// Any fx.In struct is used as an injectible set of dependencies.  Fields on
// that struct are converted into COptions using the same rules as NewCOption,
// but any struct field not convertible is ignored.
func (c *C) Use(v ...interface{}) *C {
	for _, o := range v {
		co, err := NewCOption(o)
		if err == nil {
			c.options = append(c.options, co)
			continue
		}

		if dependency, ok := arrange.IsIn(o); ok {
			c.dependencies = append(c.dependencies, dependency.Type())
			continue
		}

		c.errs = append(c.errs,
			err,
			fmt.Errorf("%s does not refer to an fx.In struct", reflect.TypeOf(v)),
		)
	}

	return c
}

// unmarshalOptions returns a slice of COptions (possibly) created from both this builder's context
// and the supplied dependencies, if any.  if the supplied dependencies slice is empty, this
// method simply returns s.options.
func (c *C) unmarshalOptions(p fx.Printer, dependencies []reflect.Value) (options []COption) {
	if len(dependencies) > 0 {
		p := arrange.NewModulePrinter(Module, p)

		// visit struct fields in dependencies, building SOptions where possible
		for _, d := range dependencies {
			arrange.VisitFields(
				d,
				func(f reflect.StructField, fv reflect.Value) arrange.VisitResult {
					if arrange.IsDependency(f, fv) {
						// ignore struct fields that aren't applicable
						// this allows callers to reuse fx.In structs for different purposes
						raw := fv.Interface()
						if co, err := NewCOption(raw); err == nil {
							p.Printf("CLIENT OPTION => %T %s", raw, f.Tag)
							options = append(options, co)
						}
					}

					return arrange.VisitContinue
				},
			)
		}

		// locally defined options execute after injected options, allowing
		// local options to override global ones
		options = append(options, c.options...)
	} else {
		// optimization: for the case with no dependencies, don't bother
		// making a copy and just return the builder's options as is
		options = c.options
	}

	return
}

// applyUnmarshal does all the heavy-lifting of creating an http.Client and applying any options.
// The returned function will always return the tuple of (*http.Client, error), and its first input
// parameter will always be a ClientIn.  Subsequent input parameters, if necessary, will be the
// dependencies supplied during building.
func (c *C) applyUnmarshal(uf func(arrange.Unmarshaler, interface{}) error) interface{} {
	return reflect.MakeFunc(
		reflect.FuncOf(
			// inputs
			append(
				[]reflect.Type{reflect.TypeOf(ClientIn{})},
				c.dependencies...,
			),

			// outputs
			[]reflect.Type{
				reflect.TypeOf((*http.Client)(nil)),
				arrange.ErrorType(),
			},

			false, // not variadic
		),
		func(inputs []reflect.Value) []reflect.Value {
			var (
				client *http.Client
				err    error
			)

			if len(c.errs) > 0 {
				err = multierr.Combine(c.errs...)
			} else {
				target := arrange.NewTarget(c.prototype)
				in := inputs[0].Interface().(ClientIn)
				err = uf(in.Unmarshaler, target.UnmarshalTo.Interface())
				if err == nil {
					factory := target.Component.Interface().(ClientFactory)
					client, err = factory.NewClient()
				}

				if err == nil {
					for _, co := range c.unmarshalOptions(in.Printer, inputs[1:]) {
						err = co(client)
						if err != nil {
							break
						}
					}
				}

				if err == nil {
					// if everything's good, ensure that idle connections are closed on shutdown
					in.Lifecycle.Append(fx.Hook{
						OnStop: func(context.Context) error {
							client.CloseIdleConnections()
							return nil
						},
					})
				}
			}

			return []reflect.Value{
				reflect.ValueOf(client),
				arrange.NewErrorValue(err),
			}
		},
	).Interface()
}

func (c *C) Unmarshal() interface{} {
	return c.applyUnmarshal(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.Unmarshal(v)
		},
	)
}

// Provide produces an fx.Provide that does the same thing as Unmarshal.  This
// is the typical way to leverage this package to create an http.Client:
//
//   v := viper.New() // setup not shown
//   fx.New(
//     arrange.Supply(v), // don't forget to supply the viper as a component!
//     arrangehttp.Client().Provide(),
//     fx.Provide(
//       func(c *http.Client) MyComponent {
//         // use the client to create MyComponent
//       },
//     ),
//     fx.Invoke(
//       func(c *http.Client) {
//         // use the client as desired
//       },
//     ),
//   )
//
// Use Unmarshal instead of this method when more control over the created component
// is necessary, such as putting it in a group or naming it.
func (c *C) Provide() fx.Option {
	return fx.Provide(
		c.Unmarshal(),
	)
}

// UnmarshalKey is the same as Unmarshal, save that it unmarshals the ClientFactory from
// a specific configuration key.
func (c *C) UnmarshalKey(key string) interface{} {
	return c.applyUnmarshal(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.UnmarshalKey(key, v)
		},
	)
}

// ProvideKey unmarshals the ClientFactory from a particular Viper key.  The *http.Client
// component is named the same as that key.
//
//   v := viper.New()
//
//   type ClientIn struct {
//     fx.In
//     Client *http.Client `name:"clients.main"` // note that this name is the same as the key
//   }
//
//   fx.New(
//     arrange.Supply(v),
//     arrangehttp.Server().ProvideKey("clients.main"),
//     fx.Invoke(
//       func(in ClientIn) error {
//         // in.Client will hold the provided *http.Client
//       },
//     ),
//   )
func (c *C) ProvideKey(key string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: c.UnmarshalKey(key),
		},
	)
}
