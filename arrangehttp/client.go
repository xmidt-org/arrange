package arrangehttp

import (
	"context"
	"net/http"
	"reflect"
	"time"

	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/roundtrip"
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
	Header    http.Header
	TLS       *arrangetls.Config
}

// NewClient produces an http.Client given these unmarshaled configuration options
func (cc ClientConfig) NewClient() (client *http.Client, err error) {
	client = &http.Client{
		Timeout: cc.Timeout,
	}

	header := httpaux.NewHeader(cc.Header)
	transport, err := cc.Transport.NewTransport(cc.TLS)
	if err == nil {
		client.Transport = roundtrip.Header(header.SetTo)(transport)
	}

	return
}

// ClientIn is the set of dependencies required to build an *http.Client component.
// A parameter of this struct type will always be the first input parameter to
// the dynamic function generated by Client().Unmarshal or Client().UnmarshalKey.
type ClientIn struct {
	fx.In

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

// Client describes how to unmarshal and configure a client
type Client struct {
	// Name is the optional name of the *http.Client component
	Name string

	// Key is the configuration key from which this client's factory
	// is unmarshaled.  If Name is not set and this field is set, then
	// this field is used by default as the component name.
	//
	// If this field is unset, unmarshaling takes place at the root
	// of the configuration.
	Key string

	// Unnamed disables the defaulting of a component name when the Key
	// field is set.  Useful when an fx.App only has one client that gets
	// unmarshaled from a key.
	//
	// When this field is true, then the *http.Client is never named regardless
	// of the other fields.
	Unnamed bool

	// ClientFactory is the prototype instance used to instantiate an *http.Client.
	// If unset, ClientConfig is used.
	//
	// If set, this instance is cloned before unmarshaling.  That means any values
	// set on it will act as defaults.
	ClientFactory ClientFactory

	// Inject is the set of dependencies used to build the client.  This is a set of
	// types that are injected when the constructor created by Provide is run.
	//
	// Injected dependencies are always applied before anything in this struct.
	Inject arrange.Inject

	// Options is the set of client options outside the enclosing fx.App that are run
	// before the client is bound to the fx.App lifecycle.  Each element of this sequence
	// must be a function with one of two signatures:
	//
	//   func(*http.Client)
	//   func(*http.Client) error
	Options arrange.Invoke

	// Middleware is the set of decorators for the http.RoundTripper that come from outside
	// the enclosing fx.App.
	//
	// Any injected middleware, via the Inject field, are applied before anything
	// in this field.
	Middleware roundtrip.Chain

	// Invoke is the optional set of functions executed as an fx.Invoke option.  These functions
	// are executed after client construction.  Each element of this sequence
	// must be a function with one of two signatures:
	//
	//   func(*http.Client)
	//   func(*http.Client) error
	//
	// If this slice is empty, client code must add at least one fx.Invoke that accepts the
	// *http.Client or else the server created by this struct will not get started.
	Invoke arrange.Invoke
}

// name returns the component name of the *http.Client.  This method returns the
// empty string if the *http.Client should be an unnamed, global component.
func (c *Client) name() string {
	switch {
	case c.Unnamed:
		return ""
	case len(c.Name) > 0:
		return c.Name
	default:
		// covers the case where both Key and Name are unset
		return c.Key
	}
}

// unmarshal handles reading in a ClientFactory's state from the arrange.Unmarshaler.
// If Key is set, this method uses UnmarshalKey.  Otherwise, Unmarshal is used.
//
// If the ClientFactory field is unset, ClientConfig{} is used.
func (c *Client) unmarshal(u arrange.Unmarshaler) (cf ClientFactory, err error) {
	prototype := c.ClientFactory
	if prototype == nil {
		prototype = ClientConfig{}
	}

	target := arrange.NewTarget(prototype)
	if len(c.Key) > 0 {
		err = u.UnmarshalKey(c.Key, target.UnmarshalTo.Interface())
	} else {
		err = u.Unmarshal(target.UnmarshalTo.Interface())
	}

	cf = target.Component.Interface().(ClientFactory)
	return
}

// configure applies the dependencies (if any) and the options and middleware supplied
// on this instance to the given *http.Client
func (c *Client) configure(in ClientIn, client *http.Client, deps []reflect.Value) (err error) {
	var (
		middleware roundtrip.Chain
		options    arrange.Invoke
	)

	arrange.VisitDependencies(
		func(d arrange.Dependency) bool {
			if d.Injected() {
				arrange.TryConvert(
					d.Value.Interface(),
					func(v roundtrip.Chain) {
						middleware = middleware.Extend(v)
					},
					func(v roundtrip.Constructor) {
						middleware = middleware.Append(v)
					},
					func(v []roundtrip.Constructor) {
						middleware = middleware.Append(v...)
					},
					func(v func(*http.Client)) {
						options = append(options, v)
					},
					func(v []func(*http.Client)) {
						for _, o := range v {
							options = append(options, o)
						}
					},
					func(v func(*http.Client) error) {
						options = append(options, v)
					},
					func(v []func(*http.Client) error) {
						for _, o := range v {
							options = append(options, o)
						}
					},
				)
			}

			return true
		},
		deps...,
	)

	middleware = middleware.Extend(c.Middleware)
	options = append(options, c.Options...)
	err = multierr.Append(
		err,
		options.Call(client),
	)

	if err == nil {
		client.Transport = middleware.Then(client.Transport)
	}

	return
}

// provide is the main workhorse that unmarshals the client factory and creates the *http.Client
func (c *Client) provide(deps []reflect.Value) (client *http.Client, err error) {
	// the first dependency is always a ClientIn
	in := deps[0].Interface().(ClientIn)

	var cf ClientFactory
	cf, err = c.unmarshal(in.Unmarshaler)
	if err != nil {
		return
	}

	client, err = cf.NewClient()
	if err != nil {
		return
	}

	err = c.configure(in, client, deps[1:])
	if err != nil {
		return
	}

	in.Lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			client.CloseIdleConnections()
			return nil
		},
	})

	return
}

// Provide creates an fx.Option that bootstraps an HTTP client.  A *http.Client
// component is returned to the enclosing fx.App.
//
// The constructor supplied to the enclosing fx.App always has a ClientIn as an
// input parameter followed by each type contained in the Inject field (if any).
// This dynamically created constructor implements a basic workflow:
//
//   - A clone of the ClientFactory object is unmarshaled.  An instance of ClientConfig
//     is used if no ClientFactory is supplied.
//
//   - The ClientFactory is invoked to create the *http.Client.
//
//   - Each injected value, dictated by the types in Inject, are examined to see if they
//     contain dependencies that apply to building a client (see below).
//
//   - Each functional option in the Inject dependencies or Options is executed with the client instance.
//
//   - Any middleware found in the Inject dependencies or Middleware are applied to the *http.Client.
//
//   - If Invoke is not empty, then an fx.Invoke option is also created that is injected with
//     the *http.Client instance created above and executes each Invoke closure.
//
// The set of dependencies in Inject that can apply to an *http.Server are very flexible:
//
//   - anything convertible to a httpaux client Constructor or Chain will decorate the *http.Client.
//
//   - any function type that takes a sole parameter of *http.Client and returns either nothing
//     or an error will be executed as a server option along with everything in the Options field.
//     This also includes slices of the same function types.
//nolint:dupl // deduping this with the client would make it less readable
func (c Client) Provide() fx.Option {
	provideFunc := arrange.Inject{reflect.TypeOf(ClientIn{})}.
		Extend(c.Inject).
		MakeFunc(c.provide)

	name := c.name()
	var options []fx.Option
	if len(name) > 0 {
		options = append(options, fx.Provide(
			fx.Annotated{
				Name:   name,
				Target: provideFunc.Interface(),
			},
		))
	} else {
		options = append(options, fx.Provide(
			provideFunc.Interface(),
		))
	}

	if len(c.Invoke) > 0 {
		var invokeFunc reflect.Value
		if len(name) > 0 {
			// build an fx.In struct
			invokeFunc = arrange.Inject{
				arrange.Struct{}.In().Append(
					arrange.Field{
						Name: name,
						Type: (*http.Client)(nil),
					},
				).Of(),
			}.MakeFunc(
				func(inputs []reflect.Value) error {
					// the router will always be the 2nd field of the only struct parameter
					return c.Invoke.Call(inputs[0].Field(1))
				},
			)
		} else {
			// just a simple global, unnamed dependency
			invokeFunc = arrange.Inject{
				(*http.Client)(nil),
			}.MakeFunc(
				func(inputs []reflect.Value) error {
					return c.Invoke.Call(inputs[0])
				},
			)
		}

		options = append(options, fx.Invoke(
			invokeFunc.Interface(),
		))
	}

	return fx.Options(options...)
}
