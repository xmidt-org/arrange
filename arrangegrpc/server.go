package arrangegrpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangelisten"
	"github.com/xmidt-org/arrange/arrangemiddle"
	"github.com/xmidt-org/arrange/arrangeoption"
	"go.uber.org/fx"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"net"
)

const (
	// ServerAbnormalExitCode is the shutdown exit code, returned by the process,
	// when an *grpc.Server exits with an error OTHER than ErrServerClosed.
	ServerAbnormalExitCode = 255
)

var (
	// ErrServerNameRequired indicates that ProvideServer or ProvideServerCustom was called
	// with an empty server name.
	ErrServerNameRequired = errors.New("A server name is required")
)

// NewServer is the primary server constructor for arrange.  Use this when you are creating a server
// from a (possibly unmarshaled) ServerConfig.  The options can be annotated to come from a value group,
// which is useful when there are multiple servers in a single fx.App.
//
// ProvideServer gives an opinionated approach to using this function to create an *grpc.Server.
// However, this function can be used by itself to allow very flexible binding:
//
//	app := fx.New(
//	  fx.Provide(
//	    arrangegrpc.NewServer, // all the parameters need to be global, unnamed components
//	    fx.Annotate(
//	      arrangegrpc.NewServer,
//	      fx.ResultTags(`name:"myserver"`), // change the component name of the *grpc.Server
//	    ),
//	  ),
//	)
func NewServer(sc ServerConfig, interceptors []grpc.UnaryServerInterceptor, serverOptions []grpc.ServerOption, opts ...arrangeoption.Option[grpc.Server]) (*grpc.Server, error) {
	return NewServerCustom(sc, interceptors, serverOptions, opts...)
}

// NewServerCustom is a server constructor that allows a client to customize the concrete
// ServerFactory and http.Handler for the server.  This function is useful when you have a
// custom (possibly unmarshaled) configuration struct that implements ServerFactory.
//
// The ServerFactory may also optionally implement Option[grpc.Server].  If it does, the factory
// option is applied after all other options have run.
func NewServerCustom[F ServerFactory](sf F, i []grpc.UnaryServerInterceptor, o []grpc.ServerOption, opts ...arrangeoption.Option[grpc.Server]) (s *grpc.Server, err error) {
	s, err = sf.NewServer(i, o...)
	if err == nil {
		s, err = arrangeoption.ApplyOptions(s, opts...)
	}

	// if the factory is itself an option, apply it last
	if fo, ok := any(sf).(arrangeoption.Option[grpc.Server]); ok && err == nil {
		err = fo.Apply(s)
	}

	return
}

// serverProvider is an internal strategy for managing a server's lifecycle within an
// enclosing fx.App.
type serverProvider[F ServerFactory] struct {
	serverName string

	// options are the externally supplied options.  These are not injected, but are
	// supplied via the ProvideXXX call.
	options []arrangeoption.Option[grpc.Server]

	// listenerMiddleware are the externally supplied listener middleware.  Similar to options.
	listenerMiddleware []arrangelisten.ListenerMiddleware

	// interceptors are the externally supplied grpc.UnaryServerInterceptor.  Similar to options.
	interceptors []grpc.UnaryServerInterceptor

	// serverOptions are the externally supplied grpc.ServerOption.  Similar to options.
	serverOptions []grpc.ServerOption
}

func newServerProvider[F ServerFactory](serverName string, external ...any) (sp serverProvider[F], err error) {
	sp.serverName = serverName
	if len(sp.serverName) == 0 {
		err = multierr.Append(err, ErrServerNameRequired)
	}

	for _, e := range external {
		if o, ok := e.(arrangeoption.Option[grpc.Server]); ok {
			sp.options = append(sp.options, o)
		} else if lm, ok := e.(func(net.Listener) net.Listener); ok {
			sp.listenerMiddleware = append(sp.listenerMiddleware, lm)
		} else if usi, ok := e.([]grpc.UnaryServerInterceptor); ok {
			sp.interceptors = append(sp.interceptors, usi...)
		} else if so, ok := e.([]grpc.ServerOption); ok {
			sp.serverOptions = append(sp.serverOptions, so...)
		} else {
			err = multierr.Append(err, fmt.Errorf("%T is not a valid external server option", e))
		}
	}

	return
}

// newServer is the server constructor function.
func (sp serverProvider[F]) newServer(sf F, i []grpc.UnaryServerInterceptor, o []grpc.ServerOption, injected ...arrangeoption.Option[grpc.Server]) (s *grpc.Server, err error) {
	s, err = NewServerCustom[F](sf, append(sp.interceptors, i...), append(sp.serverOptions, o...), injected...)
	if err == nil {
		s, err = arrangeoption.ApplyOptions(s, sp.options...)
	}

	return
}

// newListener creates a net.Listener for a given *grpc.Server.
func (sp serverProvider[F]) newListener(ctx context.Context, sf F, injected ...arrangelisten.ListenerMiddleware) (l net.Listener, err error) {
	l, err = arrangelisten.NewListener(ctx, sf, injected...)
	if err == nil {
		l = arrangemiddle.ApplyMiddleware(l, sp.listenerMiddleware...)
	}

	return
}

// runServer starts the server and ensures that the enclosing fx.App is shutdown no matter
// how the server terminates.
func (sp serverProvider[F]) runServer(sh fx.Shutdowner, s *grpc.Server, l net.Listener) {
	go arrange.ShutdownWhenDone(
		sh,
		// TODO: make the error coder configurable somehow
		func(err error) int {
			if !errors.Is(err, grpc.ErrServerStopped) {
				return ServerAbnormalExitCode
			}
			fmt.Println(err)
			return 0
		},
		func() error {
			return s.Serve(l)
		},
	)
}

// bindServer binds a server to the lifecycle of an enclosing fx.App.
func (sp serverProvider[F]) bindServer(sf F, s *grpc.Server, lc fx.Lifecycle, sh fx.Shutdowner, injected ...arrangelisten.ListenerMiddleware) {
	lc.Append(fx.StartStopHook(
		func(ctx context.Context) (err error) {
			var l net.Listener
			l, err = sp.newListener(ctx, sf, injected...)
			if err == nil {
				sp.runServer(sh, s, l)
			}

			return
		},
		s.GracefulStop,
	))
}

// ProvideServer assembles a server out of application components in a standard, opinionated way.
// The serverName parameter is used as both the name of the *grpc.Server component and a prefix
// for that server's dependencies:
//
//   - NewServer is used to create the server as a component named serverName
//   - ServerConfig is an optional dependency with the name serverName+".config"
//   - []grpc.UnaryServerInterceptor is a value group dependency with the name serverName+".interceptors"
//   - []grpc.ServerOption is a value group dependency with the name serverName+".server.options"
//   - []Option[grpc.Server] is a value group dependency with the name serverName+".options"
//   - []ListenerMiddleware is a value group dependency with the name serverName+".listener.middleware"
//
// The external slice contains items that come from outside the enclosing fx.App that are applied to
// the server and listener.  Each element of external must be either an Option[grpc.Server] or a
// ListenerMiddleware.  Any other type short circuits application startup with an error.
func ProvideServer(serverName string, external ...any) fx.Option {
	return ProvideServerCustom[ServerConfig](serverName, external...)
}

// ProvideServerCustom is like ProvideServer, but it allows customization of the concrete
// ServerFactory and http.Handler dependencies.
func ProvideServerCustom[F ServerFactory](serverName string, external ...any) fx.Option {
	sp, err := newServerProvider[F](serverName, external...)
	if err != nil {
		return fx.Error(err)
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				sp.newServer,
				arrange.Tags().Push(serverName).
					OptionalName("config").
					Group("interceptors").
					Group("server.options").
					Group("options").
					ParamTags(),
				arrange.Tags().Name(serverName).ResultTags(),
			),
		),
		fx.Invoke(
			fx.Annotate(
				sp.bindServer,
				arrange.Tags().Push(serverName).
					OptionalName("config").
					Push("").Name(serverName).Pop().
					Skip().
					Skip().
					Group("listener.middleware").
					ParamTags(),
			),
		),
	)
}
