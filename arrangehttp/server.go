package arrangehttp

import (
	"errors"
	"net"
	"net/http"

	"go.uber.org/fx"
)

// ApplyServerOptions executes options against a server.  The original server is returned, along
// with any error(s) that occurred.  All options are executed, so the returned error may be an
// aggregate error which can be inspected via go.uber.org/multierr.
//
// This function can be used as an fx decorator for a server within the enclosing application.
func ApplyServerOptions(server *http.Server, opts ...ServerOption) (*http.Server, error) {
	err := ServerOptions(opts).Apply(server)
	return server, err
}

// NewServerCustom is a server constructor that allows a client to customize the concrete
// ServerFactory and http.Handler for the server.  This function is useful when you have a
// custom (possibly unmarshaled) configuration struct that implements ServerFactory.
func NewServerCustom[F ServerFactory, H http.Handler](sf F, h H, opts ...ServerOption) (s *http.Server, err error) {
	s, err = sf.NewServer()
	if err == nil {
		s.Handler = h
		s, err = ApplyServerOptions(s, opts...)
	}

	return
}

// NewServer is the primary server constructor for arrange.  Use this when you are creating a server
// from a (possibly unmarshaled) ServerConfig.  The options can be annotated to come from a value group,
// which is useful when there are multiple servers in a single fx.App.
func NewServer(sc ServerConfig, h http.Handler, opts ...ServerOption) (*http.Server, error) {
	return NewServerCustom(sc, h, opts...)
}

func serve(server *http.Server, listener net.Listener, shutdowner fx.Shutdowner) {
	defer shutdowner.Shutdown()
	err := server.Serve(listener)
	if !errors.Is(err, http.ErrServerClosed) {
		// TODO
	}
}

func listenAndServeTLS(server *http.Server, shutdowner fx.Shutdowner) {
	defer shutdowner.Shutdown()
	err := server.ListenAndServeTLS("", "") // certificates must be in the TLSConfig
	if !errors.Is(err, http.ErrServerClosed) {
		// TODO
	}
}

func listenAndServe(server *http.Server, shutdowner fx.Shutdowner) {
	defer shutdowner.Shutdown()
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		// TODO
	}
}

func newServerHook(server *http.Server, listener net.Listener, shutdowner fx.Shutdowner) (hook fx.Hook) {
	switch {
	case listener != nil:
		hook = fx.StartHook(
			func() {
				go serve(server, listener, shutdowner)
			},
		)

	case server.TLSConfig != nil:
		hook = fx.StartHook(
			func() {
				go listenAndServeTLS(server, shutdowner)
			},
		)

	default:
		hook = fx.StartHook(
			func() {
				go listenAndServe(server, shutdowner)
			},
		)
	}

	hook.OnStop = server.Shutdown
	return
}

// BindServer binds a server to the enclosing application's lifecycle.
//
// - If listener is not nil, then the server is started with http.Server.Serve.
// - If server.TLSConfig is not nil, then a TLS listener is created and the server is started via http.Server.ServeTLS.
// - Finally, by default the server is started with http.Server.ListenAndServe.
//
// The server is shutdown gracefully via http.Server.Shutdown.
func BindServer(server *http.Server, listener net.Listener, lifecycle fx.Lifecycle, shutdowner fx.Shutdowner) {
	lifecycle.Append(
		newServerHook(server, listener, shutdowner),
	)
}
