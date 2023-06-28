package arrangehttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"reflect"

	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/internal/arrangereflect"
	"go.uber.org/fx"
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
func NewServerCustom[F ServerFactory, H http.Handler](sf F, h H, opts ...Option[http.Server]) (s *http.Server, err error) {
	s, err = sf.NewServer()
	if err == nil {
		// guard against both the http.Handler being nil and it being
		// a non-nil interface tuple that points to a nil instance.
		// this allows all types of handlers to be optional components.
		hv := reflect.ValueOf(h)
		if !hv.IsValid() || (hv.Kind() == reflect.Ptr && hv.IsNil()) {
			s.Handler = http.DefaultServeMux
		} else {
			s.Handler = h
		}

		s, err = ApplyOptions(s, opts...)
	}

	// if the factory is itself an option, apply it last
	if fo, ok := any(sf).(Option[http.Server]); ok && err == nil {
		err = fo.Apply(s)
	}

	return
}

// ProvideServer assembles a server out of application components in a standard, opinionated way.
// The serverName parameter is used as both the name of the *http.Server component and a prefix
// for that server's dependencies:
//
//   - NewServer is used to create the server as a component named serverName
//   - ServerConfig is an optional dependency with the name serverName+".config"
//   - http.Handler is an optional dependency with the name serverName+".handler"
//   - []Option[http.Server] is a value group dependency with the name serverName+".options"
//
// The external set of options, if supplied, is applied to the server after any injected options.
// This allows for options that come from outside the enclosing fx.App, as might be the case
// for options driven by the command line.
func ProvideServer(serverName string, external ...Option[http.Server]) fx.Option {
	return ProvideServerCustom[ServerConfig, http.Handler](serverName, external...)
}

// ProvideServerCustom is like ProvideServer, but it allows customization of the concrete
// ServerFactory and http.Handler dependencies.
//
// If the concrete ServerFactory type also implements ListenerFactory, it is used to create
// the net.Listener for the server.  Otherwise, DefaultListenerFactory is used.
func ProvideServerCustom[F ServerFactory, H http.Handler](serverName string, external ...Option[http.Server]) fx.Option {
	if len(serverName) == 0 {
		return fx.Error(ErrServerNameRequired)
	}

	// Use the named constructor function when possible so that uber/fx's error reporting
	// will call out that function in logs.
	ctor := NewServerCustom[F, H]
	if len(external) > 0 {
		ctor = func(sf F, h H, injected ...Option[http.Server]) (s *http.Server, err error) {
			s, err = NewServerCustom(sf, h, injected...)
			if err == nil {
				s, err = ApplyOptions(s, external...)
			}

			return
		}
	}

	return fx.Provide(
		fx.Annotate(
			ctor,
			arrange.Tags().Push(serverName).
				OptionalName("config").
				OptionalName("handler").
				Group("options").
				ParamTags(),
			arrange.Tags().Name(serverName).ResultTags(),
		),
	)
}

// BindServer binds a server to the enclosing application's lifecycle.  The ServerConfig, acting as a
// ListenerFactory, is used to create the listener.  Middleware is applied to this listener before
// calling http.Server.Serve.
//
// The server is shutdown gracefully via http.Server.Shutdown.
//
// InvokeServer provides an opinionated way to use this function.  However, this function can be
// used with fx.Invoke directly to allow very flexible ways of binding a server:
//
//	app := fx.New(
//	  fx.Invoke(
//	    arrangehttp.BindServer, // all the parameters need to be global
//	    fx.Annotate(
//	      arrangehttp.BindServer,
//	      fx.ParamTags(
//	        "", // the ServerConfig is a global component
//	        `name:"myserver"`, // the name of the *http.Server being bound
//	      ),
//	    ),
//	  ),
//	)
func BindServer(cfg ServerConfig, server *http.Server, lifecycle fx.Lifecycle, shutdowner fx.Shutdowner, lm ...ListenerMiddleware) {
	BindServerCustom(cfg, server, lifecycle, shutdowner, lm...)
}

// BindServerCustom is like BindServer, but allows injection of a different concrete type for the ListenerFactory.
func BindServerCustom[F ListenerFactory](cfg F, server *http.Server, lifecycle fx.Lifecycle, shutdowner fx.Shutdowner, lm ...ListenerMiddleware) {
	lifecycle.Append(
		fx.StartStopHook(
			func(ctx context.Context) (err error) {
				lf := arrangereflect.Safe[ListenerFactory](cfg, DefaultListenerFactory{})

				var l net.Listener
				l, err = lf.Listen(ctx, server)
				if err == nil {
					l = ApplyMiddleware(l, lm...)
					go func() {
						var exitCode int
						defer func() {
							shutdowner.Shutdown(
								fx.ExitCode(exitCode),
							)
						}()

						serveErr := server.Serve(l)
						if !errors.Is(serveErr, http.ErrServerClosed) {
							exitCode = ServerAbnormalExitCode
						}
					}()
				}

				return
			},
			server.Shutdown,
		),
	)
}

func InvokeServer(serverName string, external ...ListenerMiddleware) fx.Option {
	return InvokeServerCustom[ServerConfig](serverName, external...)
}

func InvokeServerCustom[F ListenerFactory](serverName string, external ...ListenerMiddleware) fx.Option {
	if len(serverName) == 0 {
		return fx.Error(ErrServerNameRequired)
	}

	invoke := BindServerCustom[F]
	if len(external) > 0 {
		invoke = func(lf F, server *http.Server, lifecycle fx.Lifecycle, shutdowner fx.Shutdowner, injected ...ListenerMiddleware) {
			m := make([]ListenerMiddleware, 0, len(injected)+len(external))
			m = append(m, external...) // external middleware will execute first, before injected
			m = append(m, injected...)
			BindServerCustom(lf, server, lifecycle, shutdowner, m...)
		}
	}

	return fx.Invoke(
		fx.Annotate(
			invoke,
			arrange.Tags().Push(serverName).
				OptionalName("config").
				Push("").Name(serverName).Pop().
				Skip().
				Skip().
				Group("listener.middleware").
				ParamTags(),
		),
	)
}
