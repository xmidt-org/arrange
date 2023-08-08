package arrangegrpc

import (
	"errors"
	"fmt"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangelisten"
	"github.com/xmidt-org/arrange/arrangeoption"
	"go.uber.org/fx"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"net"
)

var (
	// ErrClientNameRequired indicates that ProvideClient or ProvideClientCustom was called
	// with an empty client name.
	ErrClientNameRequired = errors.New("A client name is required")
)

// NewClient is the primary client constructor for arrange.  Use this when you are creating a client
// from a (possibly unmarshaled) ClientConfig.  The options can be annotated to come from a value group,
// which is useful when there are multiple clients in a single fx.App.
//
// ProvideClient gives an opinionated approach to using this function to create an *grpc.ClientConn.
// However, this function can be used by itself to allow very flexible binding:
//
//	app := fx.New(
//	  fx.Provide(
//	    arrangegrpc.NewClient, // all the parameters need to be global, unnamed components
//	    fx.Annotate(
//	      arrangegrpc.NewClient,
//	      fx.ResultTags(`name:"myclient"`), // change the component name of the *grpc.ClientConn
//	    ),
//	  ),
//	)
func NewClient(sc ClientConfig, interceptors []grpc.UnaryClientInterceptor, DialOptions []grpc.DialOption, opts ...arrangeoption.Option[grpc.ClientConn]) (*grpc.ClientConn, error) {
	return NewClientCustom(sc, interceptors, DialOptions, opts...)
}

// NewClientCustom is a client constructor that allows a client to customize the concrete
// ClientFactory and http.Handler for the client.  This function is useful when you have a
// custom (possibly unmarshaled) configuration struct that implements ClientFactory.
//
// The ClientFactory may also optionally implement Option[grpc.ClientConn].  If it does, the factory
// option is applied after all other options have run.
func NewClientCustom[F ClientFactory](sf F, i []grpc.UnaryClientInterceptor, o []grpc.DialOption, opts ...arrangeoption.Option[grpc.ClientConn]) (s *grpc.ClientConn, err error) {
	s, err = sf.NewClient(i, o...)
	if err == nil {
		s, err = arrangeoption.ApplyOptions(s, opts...)
	}

	// if the factory is itself an option, apply it last
	if fo, ok := any(sf).(arrangeoption.Option[grpc.ClientConn]); ok && err == nil {
		err = fo.Apply(s)
	}

	return
}

// clientProvider is an internal strategy for managing a client's lifecycle within an
// enclosing fx.App.
type clientProvider[F ClientFactory] struct {
	clientName string

	// options are the externally supplied options.  These are not injected, but are
	// supplied via the ProvideXXX call.
	options []arrangeoption.Option[grpc.ClientConn]

	// listenerMiddleware are the externally supplied listener middleware.  Similar to options.
	listenerMiddleware []arrangelisten.ListenerMiddleware

	// interceptors are the externally supplied grpc.UnaryClientInterceptor.  Similar to options.
	interceptors []grpc.UnaryClientInterceptor

	// dialOptions are the externally supplied grpc.DialOption.  Similar to options.
	dialOptions []grpc.DialOption
}

func newClientProvider[F ClientFactory](clientName string, external ...any) (sp clientProvider[F], err error) {
	sp.clientName = clientName
	if len(sp.clientName) == 0 {
		err = multierr.Append(err, ErrClientNameRequired)
	}

	for _, e := range external {
		if o, ok := e.(arrangeoption.Option[grpc.ClientConn]); ok {
			sp.options = append(sp.options, o)
		} else if lm, ok := e.(func(net.Listener) net.Listener); ok {
			sp.listenerMiddleware = append(sp.listenerMiddleware, lm)
		} else if usi, ok := e.([]grpc.UnaryClientInterceptor); ok {
			sp.interceptors = append(sp.interceptors, usi...)
		} else if so, ok := e.([]grpc.DialOption); ok {
			sp.dialOptions = append(sp.dialOptions, so...)
		} else {
			err = multierr.Append(err, fmt.Errorf("%T is not a valid external client option", e))
		}
	}

	return
}

// newClient is the client constructor function.
func (sp clientProvider[F]) newClient(sf F, i []grpc.UnaryClientInterceptor, o []grpc.DialOption, injected ...arrangeoption.Option[grpc.ClientConn]) (s *grpc.ClientConn, err error) {
	s, err = NewClientCustom[F](sf, i, o, injected...)
	if err == nil {
		s, err = arrangeoption.ApplyOptions(s, sp.options...)
	}

	return
}

// ProvideClient assembles a client out of application components in a standard, opinionated way.
// The clientName parameter is used as both the name of the *grpc.ClientConn component and a prefix
// for that client's dependencies:
//
//   - NewClient is used to create the client as a component named clientName
//   - ClientConfig is an optional dependency with the name clientName+".config"
//   - []grpc.UnaryClientInterceptor is a value group dependency with the name clientName+".interceptors"
//   - []grpc.DialOption is a value group dependency with the name clientName+".dial.options"
//   - []arrangeoption.Option[grpc.ClientConn] is a value group dependency with the name clientName+".options"
//
// The external slice contains items that come from outside the enclosing fx.App that are applied to
// the clientconn.  Each element of external must be either an arrangeoption.Option[grpc.ClientConn],
// []grpc.UnaryClientInterceptor or []grpc.DialOption.  Any other type short circuits application startup with an error.
func ProvideClient(clientName string, external ...any) fx.Option {
	return ProvideClientCustom[ClientFactory](clientName, external...)
}

// ProvideClientCustom is like ProvideClient, but it allows customization of the concrete
// ClientFactory dependency.
func ProvideClientCustom[F ClientFactory](clientName string, external ...any) fx.Option {
	sp, err := newClientProvider[F](clientName, external...)
	if err != nil {
		return fx.Error(err)
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				sp.newClient,
				arrange.Tags().Push(clientName).
					OptionalName("config").
					Group("interceptors").
					Group("dial.options").
					Group("options").
					ParamTags(),
				arrange.Tags().Name(clientName).ResultTags(),
			),
		),
	)
}
