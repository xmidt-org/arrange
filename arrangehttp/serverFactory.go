package arrangehttp

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/xmidt-org/arrange/arrangetls"
)

// ServerFactory is the strategy for instantiating an *http.Server.  ServerConfig is this
// package's implementation of this interface, and allows a ServerFactory instance to be
// read from an external source.
//
// A custom ServerFactory implementation can be injected and used via NewServerCustom
// or ProvideServerCustom.
type ServerFactory interface {
	NewServer() (*http.Server, error)
}

// ServerConfig is the built-in ServerFactory implementation for this package.
// This struct can be unmarshaled from an external source, or supplied literally
// to the *fx.App.
type ServerConfig struct {
	// Network is the tcp network to listen on.  The default is "tcp".
	Network string `json:"network" yaml:"network"`

	// Address is the bind address of the server.  If unset, the server binds to
	// the first port available.  In that case, CaptureListenAddress can be used
	// to obtain the bind address for the server.
	Address string `json:"address" yaml:"address"`

	// ReadTimeout corresponds to http.Server.ReadTimeout
	ReadTimeout time.Duration `json:"readTimeout" yaml:"readTimeout"`

	// ReadHeaderTimeout corresponds to http.Server.ReadHeaderTimeout
	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout" yaml:"readHeaderTimeout"`

	// WriteTime corresponds to http.Server.WriteTimeout
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// IdleTimeout corresponds to http.Server.IdleTimeout
	IdleTimeout time.Duration `json:"idleTimeout" yaml:"idleTimeout"`

	// MaxHeaderBytes corresponds to http.Server.MaxHeaderBytes
	MaxHeaderBytes int `json:"maxHeaderBytes" yaml:"maxHeaderBytes"`

	// KeepAlive corresponds to net.ListenConfig.KeepAlive.  This value is
	// only used for listeners created via Listen.
	KeepAlive time.Duration `json:"keepAlive" yaml:"keepAlive"`

	// Header supplies HTTP headers to emit on every response from this server
	Header http.Header `json:"header" yaml:"header"`

	// TLS is the optional unmarshaled TLS configuration.  If set, the resulting
	// server will use HTTPS.
	TLS *arrangetls.Config `json:"tls" yaml:"tls"`
}

// NewServer is the built-in implementation of ServerFactory in this package.
// This should serve most needs.  Nothing needs to be done to use this implementation.
// By default, a Fluent Builder chain begun with Server() will use ServerConfig.
func (sc ServerConfig) NewServer() (server *http.Server, err error) {
	server = &http.Server{
		Addr:              sc.Address,
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		WriteTimeout:      sc.WriteTimeout,
		IdleTimeout:       sc.IdleTimeout,
		MaxHeaderBytes:    sc.MaxHeaderBytes,
	}

	server.TLSConfig, err = sc.TLS.New()
	return
}

// Listen is the ListenerFactory implementation driven by ServerConfig
func (sc ServerConfig) Listen(ctx context.Context, s *http.Server) (net.Listener, error) {
	return DefaultListenerFactory{
		ListenConfig: net.ListenConfig{
			KeepAlive: sc.KeepAlive,
		},
		Network: sc.Network,
	}.Listen(ctx, s)
}

type headerDecorator struct {
	names  []string
	values []string
	next   http.Handler
}

func (hd headerDecorator) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	for i := 0; i < len(hd.names); i++ {
		response.Header().Add(hd.names[i], hd.values[i])
	}

	hd.next.ServeHTTP(response, request)
}

func newHeaderDecorator(h http.Header, next http.Handler) (hd headerDecorator) {
	hd.names = make([]string, 0, len(h))
	hd.values = make([]string, 0, len(h))
	hd.next = next

	for name, values := range h {
		for _, value := range values {
			hd.names = append(hd.names, name)
			hd.values = append(hd.values, value)
		}
	}

	return
}

// Apply allows this configuration object to be seen as an Option[http.Server].
// This method adds the configured headers to every response.
func (sc ServerConfig) Apply(s *http.Server) error {
	if len(sc.Header) > 0 {
		return ServerMiddleware(func(next http.Handler) http.Handler {
			return newHeaderDecorator(sc.Header, next)
		}).Apply(s)
	}

	return nil
}
