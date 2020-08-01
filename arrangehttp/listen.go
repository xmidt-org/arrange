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
// given server has a tls.Config set, the returned listener will create TLS connections
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

// ListenConstructor is a decorator for Listen closures.  A constructor may choose
// to decorate the returned net.Listener and/or interact with other infrastructure
// when the net.Listener is built.
type ListenConstructor func(Listen) Listen

// ListenChain is a sequence of ListenConstructors.  A ListenChain is immutable,
// and will apply its constructors in order.  The zero value for this type is a valid,
// empty chain that will not decorate anything.
type ListenChain struct {
	c []ListenConstructor
}

// NewListenChain creates a chain from a sequence of constructors.  The constructors
// are always applied in the order presented here.
func NewListenChain(c ...ListenConstructor) ListenChain {
	return ListenChain{
		c: append([]ListenConstructor{}, c...),
	}
}

// Append adds additional ListenConstructors to this chain, and returns the new chain.
// This chain is not modified.  If more has zero length, this chain is returned.
func (lc ListenChain) Append(more ...ListenConstructor) ListenChain {
	if len(more) > 0 {
		return ListenChain{
			c: append(
				append([]ListenConstructor{}, lc.c...),
				more...,
			),
		}
	}

	return lc
}

// Extend is like Append, except that the additional ListenConstructors come from
// another chain
func (lc ListenChain) Extend(more ListenChain) ListenChain {
	return lc.Append(more.c...)
}

// Then decorates the given Listen strategy with all of the constructors
// applied, in the order they were presented to this chain.
func (lc ListenChain) Then(next Listen) Listen {
	// apply in reverse order, so that the order of
	// execution matches the order supplied to this chain
	for i := len(lc.c); i >= 0; i-- {
		next = lc.c[i](next)
	}

	return next
}

// CaptureAddr returns a ListenConstructor that sends the actual network address of
// the created listener to a channel.  This is useful to capture the actual address
// of a server, usually for testing, when an address such as ":0" is used.
func CaptureAddr(ch chan<- net.Addr) ListenConstructor {
	return func(next Listen) Listen {
		return func(ctx context.Context, server *http.Server) (net.Listener, error) {
			listener, err := next(ctx, server)
			if err == nil {
				ch <- listener.Addr()
			}

			return listener, err
		}
	}
}

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
