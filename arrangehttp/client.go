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

// TransportConfig holds the unmarshalable configuration options for building an http.Transport.
// For consistency with ServerConfig, this type does not contain any TLS information.  Rather,
// TLS information must be passed to it via NewTransport.
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

func (c *C) Provide(opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		c.Unmarshal(opts...),
	)
}

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

func (c *C) ProvideKey(key string, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: c.UnmarshalKey(key, opts...),
		},
	)
}
