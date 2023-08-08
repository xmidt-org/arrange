package arrangegrpc

import (
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var (
	// ErrClientTargetRequired indicates that NewClient was called
	// with an empty target.
	ErrClientTargetRequired = errors.New("client target is required")
)

// ClientConfig holds unmarshaled client configuration options.  It is the
// built-in ClientFactory implementation in this package.
type ClientConfig struct {
	Target string

	Authority string

	InitialConnWindowSize int32
	InitialWindowSize     int32
	MaxHeaderListSize     uint32

	MaxReceiveMessageSize int
	MaxSendMessageSize    int

	DisableRetry bool

	KeepAlive keepalive.ClientParameters
}

// ClientFactory is the interface implemented by unmarshaled configuration objects
// that produces a *grpc.ClientConn.  The default implementation of this interface is ClientConfig.
type ClientFactory interface {
	NewClient(middlewares []grpc.UnaryClientInterceptor, options ...grpc.DialOption) (*grpc.ClientConn, error)
}

// NewClient produces an *grpc.ClientConn given these unmarshaled configuration options
func (cc ClientConfig) NewClient(middlewares []grpc.UnaryClientInterceptor, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	if cc.Target == "" {
		return nil, ErrClientTargetRequired
	}

	initialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if cc.KeepAlive.Time != 0 || cc.KeepAlive.Timeout != 0 {
		initialOpts = append(initialOpts, grpc.WithKeepaliveParams(cc.KeepAlive))
	}
	if cc.Authority != "" {
		initialOpts = append(initialOpts, grpc.WithAuthority(cc.Authority))
	}
	if cc.InitialConnWindowSize != 0 {
		initialOpts = append(initialOpts, grpc.WithInitialConnWindowSize(cc.InitialConnWindowSize))
	}
	if cc.InitialWindowSize != 0 {
		initialOpts = append(initialOpts, grpc.WithInitialWindowSize(cc.InitialWindowSize))
	}
	if cc.MaxHeaderListSize != 0 {
		initialOpts = append(initialOpts, grpc.WithMaxHeaderListSize(cc.MaxHeaderListSize))
	}
	if cc.DisableRetry {
		initialOpts = append(initialOpts, grpc.WithDisableRetry())
	}

	// build the default call options
	callOptions := []grpc.CallOption{}

	if cc.MaxReceiveMessageSize != 0 {
		callOptions = append(callOptions, grpc.MaxCallRecvMsgSize(cc.MaxReceiveMessageSize))
	}
	if cc.MaxSendMessageSize != 0 {
		callOptions = append(callOptions, grpc.MaxCallSendMsgSize(cc.MaxSendMessageSize))
	}

	if len(callOptions) > 0 {
		initialOpts = append(initialOpts, grpc.WithDefaultCallOptions(callOptions...))
	}

	opts := append(initialOpts, options...)

	opts = append(opts, grpc.WithChainUnaryInterceptor(middlewares...))

	return grpc.Dial(cc.Target, opts...)
}
