package arrangegrpc

import (
	"context"
	"github.com/xmidt-org/arrange/arrangelisten"
	"github.com/xmidt-org/arrange/arrangetls"
	"net"
	"runtime/debug"
	"time"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// ServerFactory is the creation strategy for grpc.Server and the
// particular listener used for the accept loop.  This interface is implemented
// by any unmarshaled struct which hold server configuration fields.
//
// An implementation may optionally implement ListenerFactory to allow control
// over how the net.Listener for a server is created.
type ServerFactory interface {
	arrangelisten.ListenerFactory

	// NewServer is responsible for creating a grpc.Server using whatever
	// information was unmarshaled into this instance.
	NewServer(middlewares []grpc.UnaryServerInterceptor, options ...grpc.ServerOption) (*grpc.Server, error)
}

// ServerConfig is the built-in ServerFactory implementation for this package.
// This struct can be unmarshaled via Viper, thus allowing a grpc.Server to
// be bootstrapped from external configuration.
type ServerConfig struct {
	// Network is the tcp network to listen on.  The default is "tcp".
	Network string

	// Address is the bind address of the server.  If unset, the server binds to
	// the first port available.  In that case, CaptureListenAddress can be used
	// to obtain the bind address for the server.
	Address string

	// MaxReceiveMessageSize corresponds to grpc.MaxRecvMsgSize
	MaxReceiveMessageSize int

	// MaxSendMessageSize corresponds to grpc.MaxSendMsgSize
	MaxSendMessageSize int

	// ConnectionTimeout corresponds to grpc.ConnectionTimeout
	ConnectionTimeout time.Duration

	// WriteBufferSize corresponds to grpc.WriteBufferSize
	WriteBufferSize int

	// ReadBufferSize corresponds to grpc.ReadBufferSize
	ReadBufferSize int

	// KeepAlive corresponds to net.ListenConfig.KeepAlive.  This value is
	// only used for listeners created via Listen.
	KeepAlive       time.Duration
	KeepAliveParams keepalive.ServerParameters

	TLS *arrangetls.Config
}

// NewServer is the built-in implementation of ServerFactory in this package.
// This should serve most needs.  Nothing needs to be done to use this implementation.
// By default, a Fluent Builder chain begun with Server() will use ServerConfig.
func (sc ServerConfig) NewServer(middlewares []grpc.UnaryServerInterceptor, options ...grpc.ServerOption) (*grpc.Server, error) {
	initialOpts := []grpc.ServerOption{}
	if sc.MaxReceiveMessageSize > 0 {
		initialOpts = append(initialOpts, grpc.MaxRecvMsgSize(sc.MaxReceiveMessageSize))
	}
	if sc.MaxSendMessageSize > 0 {
		initialOpts = append(initialOpts, grpc.MaxSendMsgSize(sc.MaxSendMessageSize))
	}
	if sc.ConnectionTimeout > 0 {
		initialOpts = append(initialOpts, grpc.ConnectionTimeout(sc.ConnectionTimeout))
	}
	if sc.WriteBufferSize > 0 {
		initialOpts = append(initialOpts, grpc.WriteBufferSize(sc.WriteBufferSize))
	}
	if sc.ReadBufferSize > 0 {
		initialOpts = append(initialOpts, grpc.ReadBufferSize(sc.ReadBufferSize))
	}
	if sc.KeepAliveParams != (keepalive.ServerParameters{}) {
		initialOpts = append(initialOpts, grpc.KeepaliveParams(sc.KeepAliveParams))
	}

	opts := append(initialOpts, options...)

	// print stack trace on panic
	handler := func(p interface{}) error {
		debug.PrintStack()
		return status.Errorf(codes.Internal, "panic triggered: %v", p)
	}

	// auto-recovery if we get panic
	middlewares = append(middlewares, grpcrecovery.UnaryServerInterceptor(grpcrecovery.WithRecoveryHandler(handler)))

	middleware := grpc.UnaryInterceptor(
		grpcmiddleware.ChainUnaryServer(
			middlewares...,
		),
	)

	s := grpc.NewServer(
		append(opts, middleware)...,
	)

	return s, nil
}

// Listen is the ListenerFactory implementation driven by ServerConfig
func (sc ServerConfig) Listen(ctx context.Context) (net.Listener, error) {
	dlf := arrangelisten.DefaultListenerFactory{
		ListenConfig: net.ListenConfig{
			KeepAlive: sc.KeepAlive,
		},
		Network: sc.Network,
		Address: sc.Address,
	}
	if sc.TLS != nil {
		tls, err := sc.TLS.New()
		if err != nil {
			return nil, err
		}
		dlf.TLSConfig = tls
	}
	return dlf.Listen(ctx)
}

// Apply allows this configuration object to be seen as an Option[http.Server].
// This method adds the configured headers to every response.
func (sc ServerConfig) Apply(s *grpc.Server) error {
	return nil
}
