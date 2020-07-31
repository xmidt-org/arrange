package arrangehttp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"go.uber.org/fx"
)

// Listen is a closure factory for net.Listener instances.  Since any applied
// options may have changed the http.Server instance, this strategy is passed
// that server instance.
//
// The http.Server.Addr field is use as the address of the listener.  If the
// given server has a tls.Config set, the returned listener will be a TLS listener
// with that configuration.  The server's tls.Config must be completely setup,
// with certificates and all data structures initialized.  If the server does not
// have a tls.Config set, then a standard TCP listener is used for accepts.
//
// The returned net.Listener may be decorated arbitrarily.  Callers cannot
// assume the actual type will be *net.TCPListener, although that will always
// be the ultimate listener that accepts connections.
//
// The built-in implementation of this type is ListenerFactory.Listen.
type Listen func(context.Context, *http.Server) (net.Listener, error)

// ListenerFactory is a configurable factory for net.Listener instances.  This
// type serves as a convenient built-in Listen implementation.
type ListenerFactory struct {
	// ListenConfig is the object used to create the net.Listener
	ListenConfig net.ListenConfig

	// Network is the network to listen on, which must always be a TCP network.
	// If not set, "tcp" is used.
	Network string
}

// Listen creates a net.Listener using this factory's configuration.  It is
// assignable to the Listen type.
func (lf ListenerFactory) Listen(ctx context.Context, server *http.Server) (net.Listener, error) {
	network := lf.Network
	if len(network) == 0 {
		network = "tcp"
	}

	l, err := lf.ListenConfig.Listen(ctx, network, server.Addr)
	if err != nil {
		return nil, err
	}

	if server.TLSConfig != nil {
		l = tls.NewListener(l, server.TLSConfig)
	}

	return l, nil
}

// ServerExit is callback function run when the server exits its accept loop
type ServerExit func()

// ShutdownOnExit returns a ServerExit strategy that calls the supplied
// uber/fx Shutdowner when a server exits.  This ensures that if a given server
// exits its accept loop, the entire fx.App is stopped.
func ShutdownOnExit(shutdowner fx.Shutdowner, opts ...fx.ShutdownOption) ServerExit {
	return func() {
		shutdowner.Shutdown(opts...)
	}
}

// Serve executes the given server's accept loop using the supplied net.Listener.
// This function can be run as a goroutine.
//
// Any onExit functions will be called when the server's accept loop exits.
func Serve(s *http.Server, l net.Listener, onExit ...ServerExit) error {
	defer func() {
		for _, f := range onExit {
			f()
		}
	}()

	return s.Serve(l)
}

// ServerOnStart returns an fx.Hook.OnStart closure that starts the given server's
// accept loop.
func ServerOnStart(s *http.Server, l Listen, onExit ...ServerExit) func(context.Context) error {
	return func(ctx context.Context) error {
		listener, err := l(ctx, s)
		if err != nil {
			return err
		}

		go Serve(s, listener, onExit...)
		return nil
	}
}
