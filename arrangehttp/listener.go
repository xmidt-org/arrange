package arrangehttp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

// ListenerMiddleware represents a strategy for decorating net.Listener instances.
type ListenerMiddleware func(net.Listener) net.Listener

// ListenerFactory is a strategy for creating net.Listener instances.  Since any applied
// options may have changed the http.Server instance, this strategy is passed
// that server instance.
//
// The http.Server.Addr field should used as the address of the listener.  If the
// given server has a tls.Config set, the returned listener should create TLS connections
// with that configuration.
//
// The returned net.Listener may be decorated arbitrarily.  Callers cannot
// assume the actual type will be *net.TCPListener, although that will always
// be the ultimate listener that accepts connections.
//
// The built-in implementation of this type is DefaultListenerFactory.
type ListenerFactory interface {
	// Listen creates the appropriate net.Listener, binding to a TCP address in
	// the process
	Listen(context.Context, *http.Server) (net.Listener, error)
}

// DefaultListenerFactory is the default implementation of ListenerFactory.  The
// zero value of this type is a valid factory.
type DefaultListenerFactory struct {
	// ListenConfig is the object used to create the net.Listener
	ListenConfig net.ListenConfig

	// Network is the network to listen on, which must always be a TCP network.
	// If not set, "tcp" is used.
	Network string
}

// Listen provides the default ListenerFactory behavior for this package.
// It essentially does the same thing as net/http, but allows the network
// to be configured externally and ensures that the listen address matches
// the server address.
func (f DefaultListenerFactory) Listen(ctx context.Context, server *http.Server) (net.Listener, error) {
	network := f.Network
	if len(network) == 0 {
		network = "tcp"
	}

	l, err := f.ListenConfig.Listen(ctx, network, server.Addr)
	if err != nil {
		return nil, err
	}

	if server.TLSConfig != nil {
		// clone the TLSConfig, as the stdlib does, to avoid racyness
		l = tls.NewListener(l, server.TLSConfig.Clone())
	}

	return l, nil
}
