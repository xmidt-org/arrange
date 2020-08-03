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
// that server instance.  This closure is intended to be invoked as part of
// a lifecycle hook.
//
// The http.Server.Addr field is used as the address of the listener.  If the
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

// ListenerConstructor is a decorator for net.Listener instances.  If supplied to
// a server builder, a constructor is applied after the Listen closure creates the listener.
type ListenerConstructor func(net.Listener) net.Listener

// ListenerChain is a sequence of ListenerConstructors.  A ListenerChain is immutable,
// and will apply its constructors in order.  The zero value for this type is a valid,
// empty chain that will not decorate anything.
type ListenerChain struct {
	c []ListenerConstructor
}

// NewListenerChain creates a chain from a sequence of constructors.  The constructors
// are always applied in the order presented here.
func NewListenerChain(c ...ListenerConstructor) ListenerChain {
	return ListenerChain{
		c: append([]ListenerConstructor{}, c...),
	}
}

// Append adds additional ListenerConstructors to this chain, and returns the new chain.
// This chain is not modified.  If more has zero length, this chain is returned.
func (lc ListenerChain) Append(more ...ListenerConstructor) ListenerChain {
	if len(more) > 0 {
		return ListenerChain{
			c: append(
				append([]ListenerConstructor{}, lc.c...),
				more...,
			),
		}
	}

	return lc
}

// Extend is like Append, except that the additional ListenerConstructors come from
// another chain
func (lc ListenerChain) Extend(more ListenerChain) ListenerChain {
	return lc.Append(more.c...)
}

// Then decorates the given Listen strategy with all of the constructors
// applied, in the order they were presented to this chain.
func (lc ListenerChain) Then(next net.Listener) net.Listener {
	// apply in reverse order, so that the order of
	// execution matches the order supplied to this chain
	for i := len(lc.c) - 1; i >= 0; i-- {
		next = lc.c[i](next)
	}

	return next
}

// Listen produces a Listen strategy that uses this chain to decorate the
// returned net.Listener.  Any error prevents decoration.  Any empty ListenerChain
// will return the next Listen undecorated.
func (lc ListenerChain) Listen(next Listen) Listen {
	if len(lc.c) > 0 {
		return func(ctx context.Context, server *http.Server) (net.Listener, error) {
			listener, err := next(ctx, server)
			if err == nil {
				listener = lc.Then(listener)
			}

			return listener, err
		}
	}

	return next
}

// CaptureListenAddress returns a ListenerConstructor that sends the actual network address of
// the created listener to a channel.  This is useful to capture the actual address
// of a server, usually for testing, when an address such as ":0" is used.
//
// The returned contructor performs no actual decoration.
func CaptureListenAddress(ch chan<- net.Addr) ListenerConstructor {
	return func(next net.Listener) net.Listener {
		ch <- next.Addr()
		return next
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

// ServerExit is callback function run when the server exits its accept loop.
// A ServerExit function must never panic, or server cleanup will be interrupted.
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
