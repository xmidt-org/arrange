package arrangehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/internal/arrangereflect"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

const (
	// ServerAbnormalExitCode is the shutdown exit code, returned by the process,
	// when an *http.Server exits with an error OTHER than ErrServerClosed.
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
// ProvideServer gives an opinionated approach to using this function to create an *http.Server.
// However, this function can be used by itself to allow very flexible binding:
//
//	app := fx.New(
//	  fx.Provide(
//	    arrangehttp.NewServer, // all the parameters need to be global, unnamed components
//	    fx.Annotate(
//	      arrangehttp.NewServer,
//	      fx.ResultTags(`name:"myserver"`), // change the component name of the *http.Server
//	    ),
//	  ),
//	)
func NewServer(sc ServerConfig, h http.Handler, opts ...Option[http.Server]) (*http.Server, error) {
	return NewServerCustom(sc, h, opts...)
}

// NewServerCustom is a server constructor that allows a client to customize the concrete
// ServerFactory and http.Handler for the server.  This function is useful when you have a
// custom (possibly unmarshaled) configuration struct that implements ServerFactory.
//
// The ServerFactory may also optionally implement Option[http.Server].  If it does, the factory
// option is applied after all other options have run.
func NewServerCustom[H http.Handler, F ServerFactory](sf F, h H, opts ...Option[http.Server]) (s *http.Server, err error) {
	s, err = sf.NewServer()
	if err == nil {
		s.Handler = arrangereflect.Safe[http.Handler](h, http.DefaultServeMux)
		s, err = ApplyOptions(s, opts...)
	}

	// if the factory is itself an option, apply it last
	if fo, ok := any(sf).(Option[http.Server]); ok && err == nil {
		err = fo.Apply(s)
	}

	return
}

// serverProvider is an internal strategy for managing a server's lifecycle within an
// enclosing fx.App.
type serverProvider[H http.Handler, F ServerFactory] struct {
	serverName string

	// options are the externally supplied options.  These are not injected, but are
	// supplied via the ProvideXXX call.
	options []Option[http.Server]

	// listenerMiddleware are the externally supplied listener middleware.  Similar to options.
	listenerMiddleware []ListenerMiddleware
}

func newServerProvider[H http.Handler, F ServerFactory](serverName string, external ...any) (sp serverProvider[H, F], err error) {
	sp.serverName = serverName
	if len(sp.serverName) == 0 {
		err = multierr.Append(err, ErrServerNameRequired)
	}

	for _, e := range external {
		if o, ok := e.(Option[http.Server]); ok {
			sp.options = append(sp.options, o)
		} else if lm, ok := e.(ListenerMiddleware); ok {
			sp.listenerMiddleware = append(sp.listenerMiddleware, lm)
		} else {
			err = multierr.Append(err, fmt.Errorf("%T is not a valid external server option", e))
		}
	}

	return
}

// newServer is the server constructor function.
func (sp serverProvider[H, F]) newServer(sf F, h H, injected ...Option[http.Server]) (s *http.Server, err error) {
	s, err = NewServerCustom[H, F](sf, h, injected...)
	if err == nil {
		s, err = ApplyOptions(s, sp.options...)
	}

	return
}

// newListener creates a net.Listener for a given *http.Server.
func (sp serverProvider[H, F]) newListener(ctx context.Context, sf F, s *http.Server, injected ...ListenerMiddleware) (l net.Listener, err error) {
	l, err = NewListener(ctx, sf, s, injected...)
	if err == nil {
		l = ApplyMiddleware(l, sp.listenerMiddleware...)
	}

	return
}

// runServer starts the server and ensures that the enclosing fx.App is shutdown no matter
// how the server terminates.
func (sp serverProvider[H, F]) runServer(sh fx.Shutdowner, s *http.Server, l net.Listener) {
	go arrange.ShutdownWhenDone(
		sh,
		// TODO: make the error coder configurable somehow
		func(err error) int {
			if !errors.Is(err, http.ErrServerClosed) {
				return ServerAbnormalExitCode
			}

			return 0
		},
		func() error {
			return s.Serve(l)
		},
	)
}

// bindServer binds a server to the lifecycle of an enclosing fx.App.
func (sp serverProvider[H, F]) bindServer(sf F, s *http.Server, lc fx.Lifecycle, sh fx.Shutdowner, injected ...ListenerMiddleware) {
	lc.Append(fx.StartStopHook(
		func(ctx context.Context) (err error) {
			var l net.Listener
			l, err = sp.newListener(ctx, sf, s, injected...)
			if err == nil {
				sp.runServer(sh, s, l)
			}

			return
		},
		s.Shutdown,
	))
}

// ProvideServer assembles a server out of application components in a standard, opinionated way.
// The serverName parameter is used as both the name of the *http.Server component and a prefix
// for that server's dependencies:
//
//   - NewServer is used to create the server as a component named serverName
//   - ServerConfig is an optional dependency with the name serverName+".config"
//   - http.Handler is an optional dependency with the name serverName+".handler"
//   - []Option[http.Server] is a value group dependency with the name serverName+".options"
//   - []ListenerMiddleware is a value group dependency with the name serverName+".listener.middleware"
//
// The external slice contains items that come from outside the enclosing fx.App that are applied to
// the server and listener.  Each element of external must be either an Option[http.Server] or a
// ListenerMiddleware.  Any other type short circuits application startup with an error.
func ProvideServer(serverName string, external ...any) fx.Option {
	return ProvideServerCustom[http.Handler, ServerConfig](serverName, external...)
}

// ProvideServerCustom is like ProvideServer, but it allows customization of the concrete
// ServerFactory and http.Handler dependencies.
func ProvideServerCustom[H http.Handler, F ServerFactory](serverName string, external ...any) fx.Option {
	sp, err := newServerProvider[H, F](serverName, external...)
	if err != nil {
		return fx.Error(err)
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				sp.newServer,
				arrange.Tags().Push(serverName).
					OptionalName("config").
					OptionalName("handler").
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
