package arrangehttp

import (
	"context"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
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
func (tc TransportConfig) NewTransport(t *TLS) (transport *http.Transport, err error) {
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

	transport.TLSClientConfig, err = NewTLSConfig(t)
	return
}

// ClientConfig holds unmarshaled client configuration options.  It is the
// built-in ClientFactory implementation in this package.
type ClientConfig struct {
	Timeout   time.Duration
	Transport TransportConfig
	TLS       *TLS
}

// NewClient produces an http.Client given these unmarshaled configuration options
func (cc ClientConfig) NewClient() (client *http.Client, err error) {
	client = &http.Client{
		Timeout: cc.Timeout,
	}

	client.Transport, err = cc.Transport.NewTransport(cc.TLS)
	return
}

// ClientOption is a functional option type that can mutate an http.Client
// prior to its being returned to an fx.App as a component
type ClientOption func(*http.Client) error

// ClientIn is the set of dependencies required to build an *http.Client component
type ClientIn struct {
	arrange.ProvideIn

	Lifecycle  fx.Lifecycle
	Shutdowner fx.Shutdowner
}

// C is a Fluent Builder for creating an http.Client as an uber/fx component.
// This type should be constructred with the Client function.
type C struct {
	co        []ClientOption
	prototype ClientFactory
}

// Client begins a Fluent Builder chain for constructing an http.Client from
// unmarshaled configuration and introducing that http.Client as a component
// for an enclosing uber/fx app.
func Client(opts ...ClientOption) *C {
	c := new(C)
	if len(opts) > 0 {
		// safe copy
		c.co = append([]ClientOption{}, opts...)
	}

	return c.ClientFactory(ClientConfig{})
}

// ClientFactory sets the prototype factory that is unmarshaled from Viper.
// This prototype obeys the rules of arrange.NewTarget.  By default, ClientConfig
// is used as the ClientFactory.  This build method allows a caller to use
// custom configuration.
func (c *C) ClientFactory(prototype ClientFactory) *C {
	c.prototype = prototype
	return c
}

// newClient does all the heavy lifting for creating the client, applying
// options, and binding CloseIdleConnections to the fx lifecycle.
func (c *C) newClient(f ClientFactory, in ClientIn) (*http.Client, error) {
	client, err := f.NewClient()
	if err != nil {
		return nil, err
	}

	for _, f := range c.co {
		if err := f(client); err != nil {
			return nil, err
		}
	}

	in.Lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			client.CloseIdleConnections()
			return nil
		},
	})

	return client, nil
}

// Unmarshal uses an injected Viper instance to unmarshal the ClientFactory.  That factory
// is then used to create an *http.Client.  The client's CloseIdleConnections method is
// bound to the OnStop portion of the fx.App lifecycle.
//
// This method terminates the builder chain, and must be used inside fx.Provide:
//
//   v := viper.New() // setup not shown
//   fx.New(
//     arrange.Supply(v), // don't forget to supply the viper as a component!
//     fx.Provide(
//       arrangehttp.Client().Unmarshal(),
//       func(c *http.Client) MyComponent {
//         // use the client to create MyComponent
//       },
//     ),
//     fx.Invoke(
//       func(c *http.Client) error {
//         // use the client as desired
//       },
//     ),
//   )
func (c *C) Unmarshal(opts ...viper.DecoderConfigOption) func(ClientIn) (*http.Client, error) {
	return func(in ClientIn) (*http.Client, error) {
		var (
			target = arrange.NewTarget(c.prototype)
			err    = in.Viper.Unmarshal(
				target.UnmarshalTo(),
				arrange.Merge(in.DecoderOptions, opts),
			)
		)

		if err != nil {
			return nil, err
		}

		return c.newClient(
			target.Component().(ClientFactory),
			in,
		)
	}
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
func (c *C) Provide(opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		c.Unmarshal(opts...),
	)
}

// UnmarshalKey is similar to Unmarshal, but unmarshals a particular Viper configuration
// key rather than unmarshaling from the root.
//
// Assume a yaml configuration similar to:
//
//   clients:
//     main:
//       timeout: "15s"
//       transport:
//         disableCompression: true
//         writeBufferSize: 8192
//         readBufferSize: 8192
//         forceAttemptHTTP2: true
//
//
// The corresponding UnmarshalKey declaration would be:
//
//   v := viper.New() // read in the above YAML
//   fx.New(
//     arrange.Supply(v), // don't forget to supply the viper as a component!
//     fx.Provide(
//       arrangehttp.Client().UnmarshalKey("clients.main"),
//     ),
//     fx.Invoke(
//       func(c *http.Client) error {
//         // use the client as desired
//       },
//     ),
//   )
//
// Note that UnmarshalKey simply provides a constructor, as with Unmarshal.  To name
// the component, one has to use fx.Annotated.  ProvideKey does this automatically.
func (c *C) UnmarshalKey(key string, opts ...viper.DecoderConfigOption) func(ClientIn) (*http.Client, error) {
	return func(in ClientIn) (*http.Client, error) {
		var (
			target = arrange.NewTarget(c.prototype)
			err    = in.Viper.UnmarshalKey(
				key,
				target.UnmarshalTo(),
				arrange.Merge(in.DecoderOptions, opts),
			)
		)

		if err != nil {
			return nil, err
		}

		return c.newClient(
			target.Component().(ClientFactory),
			in,
		)
	}
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
func (c *C) ProvideKey(key string, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: c.UnmarshalKey(key, opts...),
		},
	)
}
