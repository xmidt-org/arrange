package arrangehttp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"
)

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

// ListenerFactoryFunc is a closure type that implements ListenerFactory
type ListenerFactoryFunc func(context.Context, *http.Server) (net.Listener, error)

// Listen implements ListenerFactory
func (lff ListenerFactoryFunc) Listen(ctx context.Context, s *http.Server) (net.Listener, error) {
	return lff(ctx, s)
}

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

// Factory decorates a ListenerFactory so that the factory's product, net.Listener,
// is decorated with the constructors in this chain.
func (lc ListenerChain) Factory(next ListenerFactory) ListenerFactory {
	if len(lc.c) > 0 {
		return ListenerFactoryFunc(func(ctx context.Context, s *http.Server) (net.Listener, error) {
			listener, err := next.Listen(ctx, s)
			if err == nil {
				listener = lc.Then(listener)
			}

			return listener, err
		})
	}

	return next
}

// CaptureListenAddress returns a ListenerConstructor that sends the actual network address of
// the created listener to a channel.  This is useful to capture the actual address
// of a server, usually for testing, when an address such as "127.0.0.1:0" is used.
//
// The returned contructor performs no actual decoration.
func CaptureListenAddress(ch chan<- net.Addr) ListenerConstructor {
	return func(next net.Listener) net.Listener {
		ch <- next.Addr()
		return next
	}
}

// AwaitListenAddress waits for a net.Addr on a channel for a specified duration.
// If no net address appears on the channel within the timeout, the given fail function
// is called with a failure message and this function returns a nil net.Addr and false.
//
// This function is intended for tests that use CaptureListenAddress to obtain the
// address of a server started on addresses like "127.0.0.1:0".  Callers may pass t.Fatalf to fail
// the test immediately or pass t.Errorf or t.Logf to continue with the test.
// The second bool return can be used to indicate if an address was actually found on the channel.
func AwaitListenAddress(fail func(string, ...interface{}), ch <-chan net.Addr, d time.Duration) (a net.Addr, ok bool) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case a = <-ch:
		ok = true
	case <-timer.C:
		fail("No listen address returned within %s", d)
	}

	return
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
func (f DefaultListenerFactory) Listen(ctx context.Context, server *http.Server) (l net.Listener, err error) {
	network := f.Network
	if len(network) == 0 {
		network = "tcp"
	}
	if server.Addr != "" {
		l, err = f.ListenConfig.Listen(ctx, network, server.Addr)
		if err != nil {
			return nil, err
		}
	} else {
		// if address is not defined use loopback
		l, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			l, err = net.Listen("tcp6", "[::1]:0")
			if err != nil {
				return nil, err
			}
		}
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

// Servable describes the behavior of an object that implements an accept loop.
// *http.Server implements this interface.
type Servable interface {
	// Serve executes an accept loop using the given listener.  This method
	// does not return until the listener is closed.
	Serve(net.Listener) error
}

// Serve executes the given servable's accept loop using the supplied net.Listener.
// This function can be run as a goroutine.
//
// Any onExit functions will be called when the server's accept loop exits.
func Serve(s Servable, l net.Listener, onExit ...ServerExit) error {
	defer func() {
		for _, f := range onExit {
			f()
		}
	}()

	return s.Serve(l)
}

// ServerOnStart returns an fx.Hook.OnStart closure that starts the given server's
// accept loop.
func ServerOnStart(s *http.Server, f ListenerFactory, onExit ...ServerExit) func(context.Context) error {
	return func(ctx context.Context) error {
		listener, err := f.Listen(ctx, s)
		if err != nil {
			return err
		}

		go Serve(s, listener, onExit...)
		return nil
	}
}
