package arrangelisten

import (
	"context"
	"crypto/tls"
	"github.com/xmidt-org/arrange/arrangemiddle"
	"github.com/xmidt-org/arrange/internal/arrangereflect"
	"net"
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
	Listen(context.Context) (net.Listener, error)
}

// DefaultListenerFactory is the default implementation of ListenerFactory.  The
// zero value of this type is a valid factory.
type DefaultListenerFactory struct {
	// ListenConfig is the object used to create the net.Listener
	ListenConfig net.ListenConfig

	// Network is the network to listen on, which must always be a TCP network.
	// If not set, "tcp" is used.
	Network string

	// Address string
	Address string

	// TLSConfig is the TLS configuration to use for the listener.  If nil,
	// no TLS is used.
	TLSConfig *tls.Config
}

// Listen provides the default ListenerFactory behavior for this package.
// It essentially does the same thing as net/http, but allows the network
// to be configured externally and ensures that the listen address matches
// the server address.
func (f DefaultListenerFactory) Listen(ctx context.Context) (net.Listener, error) {
	network := f.Network
	if len(network) == 0 {
		network = "tcp"
	}
	//
	//// if server is nil, leverage default values in the DefaultListenerFactory
	//if server != nil {
	//	f.Address = server.Addr
	//	f.TLSConfig = server.TLSConfig.Clone()
	//}

	l, err := f.ListenConfig.Listen(ctx, network, f.Address)
	if err != nil {
		return nil, err
	}

	if f.TLSConfig != nil {
		// clone the TLSConfig, as the stdlib does, to avoid racyness
		l = tls.NewListener(l, f.TLSConfig.Clone())
	}

	return l, nil
}

// NewListener encapsulates the logic for creating a net.Listener for a server.
// This function should be called from within a start hook, typically via fx.Hook.OnStart.
//
// The ListenerFactory may be nil, in which case an instance of DefaultListenerFactory will be used.
func NewListener(ctx context.Context, lf ListenerFactory, lm ...ListenerMiddleware) (l net.Listener, err error) {
	lf = arrangereflect.Safe[ListenerFactory](lf, DefaultListenerFactory{})
	l, err = lf.Listen(ctx)
	if err == nil {
		l = arrangemiddle.ApplyMiddleware(l, lm...)
	}

	return
}
