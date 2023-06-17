package arrangehttp

import (
	"errors"
	"net"
	"net/http"
	"reflect"

	"github.com/xmidt-org/arrange"
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

// ApplyServerOptions executes options against a server.  The original server is returned, along
// with any error(s) that occurred.  All options are executed, so the returned error may be an
// aggregate error which can be inspected via go.uber.org/multierr.
//
// This function can be used as an fx decorator for a server within the enclosing application.
func ApplyServerOptions(server *http.Server, opts ...ServerOption) (*http.Server, error) {
	err := ServerOptions(opts).Apply(server)
	return server, err
}

// NewServer is the primary server constructor for arrange.  Use this when you are creating a server
// from a (possibly unmarshaled) ServerConfig.  The options can be annotated to come from a value group,
// which is useful when there are multiple servers in a single fx.App.
func NewServer(sc ServerConfig, h http.Handler, opts ...ServerOption) (*http.Server, error) {
	return NewServerCustom(sc, h, opts...)
}

// NewServerCustom is a server constructor that allows a client to customize the concrete
// ServerFactory and http.Handler for the server.  This function is useful when you have a
// custom (possibly unmarshaled) configuration struct that implements ServerFactory.
func NewServerCustom[F ServerFactory, H http.Handler](sf F, h H, opts ...ServerOption) (s *http.Server, err error) {
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

		s, err = ApplyServerOptions(s, opts...)
	}

	return
}

func serve(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

func listenAndServeTLS(server *http.Server, _ net.Listener) error {
	return server.ListenAndServeTLS("", "") // certificates must be in the TLSConfig
}

func listenAndServe(server *http.Server, _ net.Listener) error {
	return server.ListenAndServe()
}

func newServerHook(server *http.Server, listener net.Listener, shutdowner fx.Shutdowner) (hook fx.Hook) {
	lv := reflect.ValueOf(listener)

	var startFunc func(*http.Server, net.Listener) error

	switch {
	// handle the case of a non-nil interface which is invalid (nil object underneath)
	case lv.IsValid() && (lv.Kind() != reflect.Ptr || !lv.IsNil()):
		startFunc = serve

	case server.TLSConfig != nil:
		startFunc = listenAndServeTLS

	default:
		startFunc = listenAndServe
	}

	hook = fx.StartStopHook(
		func() {
			go func() {
				var exitCode int
				defer func() {
					shutdowner.Shutdown(
						fx.ExitCode(exitCode),
					)
				}()

				err := startFunc(server, listener)
				if !errors.Is(err, http.ErrServerClosed) {
					exitCode = ServerAbnormalExitCode
				}
			}()
		},
		server.Shutdown,
	)

	return
}

// BindServer binds a server to the enclosing application's lifecycle.
//
// - If listener is not nil, then the server is started with http.Server.Serve.
// - If server.TLSConfig is not nil, then a TLS listener is created and the server is started via http.Server.ServeTLS.
// - Finally, by default the server is started with http.Server.ListenAndServe.
//
// The server is shutdown gracefully via http.Server.Shutdown.
func BindServer(server *http.Server, listener net.Listener, lifecycle fx.Lifecycle, shutdowner fx.Shutdowner) {
	lifecycle.Append(
		newServerHook(server, listener, shutdowner),
	)
}

// ProvideServer assembles a server out of application components in a standard, opinionated way.
// The serverName parameter is used as both the name of the *http.Server component and a prefix
// for that server's dependencies:
//
// (1) NewServer is used to create the server as a component named serverName
// (2) ServerConfig is an optional dependency with the name serverName+".config".  Making
// this optional allows the provided server to take the package defaults for configuration.
// (3) http.Handler is an optional dependency with the name serverName+".handler".  If not supplied,
// http.DefaultServeMux is used, in keeping with the behavior of net/http.
// (4) []ServerOption is an optional value group dependency with the name serverName+".options"
// (5) net.Listener is an optional dependency with the name serverName+".listener"
//
// The external set of options, if supplied, is applied to the server after any injected options.
// This allows for options that come from outside the enclosing fx.App, as might be the case
// for options driven by the command line.
//
// BindServer is used as an fx.Invoke function to bind the resulting server to the enclosing
// application's lifecycle.
func ProvideServer(serverName string, external ...ServerOption) fx.Option {
	return ProvideServerCustom[ServerConfig, http.Handler](serverName, external...)
}

// ProvideServerCustom is like ProvideServer, but it allows customization of the concrete
// ServerFactory and http.Handler dependencies.
func ProvideServerCustom[F ServerFactory, H http.Handler](serverName string, external ...ServerOption) fx.Option {
	if len(serverName) == 0 {
		return fx.Error(ErrServerNameRequired)
	}

	// Use the named constructor function when possible so that uber/fx's error reporting
	// will call out that function in logs.
	ctor := NewServerCustom[F, H]
	if len(external) > 0 {
		ctor = func(sf F, h H, injected ...ServerOption) (s *http.Server, err error) {
			s, err = NewServerCustom(sf, h, injected...)
			if err == nil {
				s, err = ApplyServerOptions(s, external...)
			}

			return
		}
	}

	prefix := serverName + "."
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				ctor,
				arrange.Tags().
					OptionalName(prefix+"config").
					OptionalName(prefix+"handler").
					Group(prefix+"options").
					OptionalName(prefix+"listener").
					ParamTags(),
				arrange.Tags().Name(serverName).ResultTags(),
			),
		),
		fx.Invoke(
			fx.Annotate(
				BindServer,
				arrange.Tags().
					Name(serverName).
					OptionalName(prefix+"listener").
					ParamTags(),
			),
		),
	)
}
