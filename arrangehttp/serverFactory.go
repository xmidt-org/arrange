package arrangehttp

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/xmidt-org/arrange/arrangetls"
)

type ServerFactory interface {
	NewServer() (*http.Server, error)
}

// ServerConfig is the built-in ServerFactory implementation for this package.
// This struct can be unmarshaled via Viper, thus allowing an http.Server to
// be bootstrapped from external configuration.
type ServerConfig struct {
	// Network is the tcp network to listen on.  The default is "tcp".
	Network string

	// Address is the bind address of the server.  If unset, the server binds to
	// the first port available.  In that case, CaptureListenAddress can be used
	// to obtain the bind address for the server.
	Address string

	// ReadTimeout corresponds to http.Server.ReadTimeout
	ReadTimeout time.Duration

	// ReadHeaderTimeout corresponds to http.Server.ReadHeaderTimeout
	ReadHeaderTimeout time.Duration

	// WriteTime corresponds to http.Server.WriteTimeout
	WriteTimeout time.Duration

	// IdleTimeout corresponds to http.Server.IdleTimeout
	IdleTimeout time.Duration

	// MaxHeaderBytes corresponds to http.Server.MaxHeaderBytes
	MaxHeaderBytes int

	// KeepAlive corresponds to net.ListenConfig.KeepAlive.  This value is
	// only used for listeners created via Listen.
	KeepAlive time.Duration

	// Header supplies HTTP headers to emit on every response from this server
	Header http.Header

	// TLS is the optional unmarshaled TLS configuration.  If set, the resulting
	// server will use HTTPS.
	TLS *arrangetls.Config
}

// NewServer is the built-in implementation of ServerFactory in this package.
// This should serve most needs.  Nothing needs to be done to use this implementation.
// By default, a Fluent Builder chain begun with Server() will use ServerConfig.
func (sc ServerConfig) NewServer() (server *http.Server, err error) {
	// TODO: header := httpaux.NewHeader(sc.Header)

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
