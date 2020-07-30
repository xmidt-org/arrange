package arrangehttp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

// Listen is a closure factory for net.Listener instances.  Since any applied
// options may have changed the http.Server instance, this strategy is passed
// that server instance.
type Listen func(context.Context, *http.Server) (net.Listener, error)

// ListenerFactory is a configurable factory for net.Listener instances.  This
// type serves as a convenient built-in Listen implementation.
type ListenerFactory struct {
	net.ListenConfig
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
