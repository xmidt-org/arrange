package arrangehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
	"github.com/xmidt-org/httpaux"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// ServerFactory is the creation strategy for both an http.Server and the
// particular listener used for the accept loop.  This interface is implemented
// by any unmarshaled struct which hold server configuration fields.
//
// An implementation may optionally implement ListenerFactory to allow control
// over how the net.Listener for a server is created.
type ServerFactory interface {
	// NewServer is responsible for creating an http.Server using whatever
	// information was unmarshaled into this instance.  The supplied http.Handler
	// is used as http.Server.Handler, though implementations are free to
	// decorate it arbitrarily.
	NewServer(http.Handler) (*http.Server, error)
}

// ServerConfig is the built-in ServerFactory implementation for this package.
// This struct can be unmarshaled via Viper, thus allowing an http.Server to
// be bootstrapped from external configuration.
type ServerConfig struct {
	// Network is the tcp network to listen on.  The default is "tcp".
	Network string

	// Address is the bind address of the server.  If unset, the server binds to
	// the first port available.  In that case, CaptureListenAddress can be used
	// to obtain the bind address for the server.
	Address string

	// ReadTimeout corresponds to http.Server.ReadTimeout
	ReadTimeout time.Duration

	// ReadHeaderTimeout corresponds to http.Server.ReadHeaderTimeout
	ReadHeaderTimeout time.Duration

	// WriteTime corresponds to http.Server.WriteTimeout
	WriteTimeout time.Duration

	// IdleTimeout corresponds to http.Server.IdleTimeout
	IdleTimeout time.Duration

	// MaxHeaderBytes corresponds to http.Server.MaxHeaderBytes
	MaxHeaderBytes int

	// KeepAlive corresponds to net.ListenConfig.KeepAlive.  This value is
	// only used for listeners created via Listen.
	KeepAlive time.Duration

	// Header supplies HTTP headers to emit on every response from this server
	Header http.Header

	// TLS is the optional unmarshaled TLS configuration.  If set, the resulting
	// server will use HTTPS.
	TLS *arrangetls.Config
}

// NewServer is the built-in implementation of ServerFactory in this package.
// This should serve most needs.  Nothing needs to be done to use this implementation.
// By default, a Fluent Builder chain begun with Server() will use ServerConfig.
func (sc ServerConfig) NewServer(h http.Handler) (server *http.Server, err error) {
	header := httpaux.NewHeader(sc.Header)

	server = &http.Server{
		Addr:              sc.Address,
		Handler:           header.Then(h),
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		WriteTimeout:      sc.WriteTimeout,
		IdleTimeout:       sc.IdleTimeout,
		MaxHeaderBytes:    sc.MaxHeaderBytes,
	}

	server.TLSConfig, err = sc.TLS.New()
	return
}

// Listen is the ListenerFactory implementation driven by ServerConfig
func (sc ServerConfig) Listen(ctx context.Context, s *http.Server) (net.Listener, error) {
	return DefaultListenerFactory{
		ListenConfig: net.ListenConfig{
			KeepAlive: sc.KeepAlive,
		},
		Network: sc.Network,
	}.Listen(ctx, s)
}

// ServerIn describes the set of dependencies for creating a mux.Router and,
// by extension, an http.Server.
type ServerIn struct {
	fx.In

	// Unmarshaler is the required arrange Unmarshaler component used to unmarshal
	// a ServerFactory
	Unmarshaler arrange.Unmarshaler

	// Printer is the optional fx.Printer used to output informational messages about
	// server unmarshaling and configuration.  If unset, arrange.DefaultPrinter() is used.
	Printer fx.Printer `optional:"true"`

	// Lifecycle is the required uber/fx Lifecycle to which the server will be bound.
	// The server will start with the app starts and will gracefully shutdown when
	// the app is stopped.
	Lifecycle fx.Lifecycle

	// Shutdowner is used to guarantee that any server which aborts its accept loop
	// will stop the entire app.
	Shutdowner fx.Shutdowner
}

// S is a Fluent Builder for unmarshaling an http.Server.  This type must be
// created with the Server function.
type S struct {
	errs         []error
	options      []sOption
	dependencies []reflect.Type
	prototype    ServerFactory
}

// Server starts a Fluent Builder method chain for creating an http.Server,
// binding its lifecycle to the fx.App lifecycle, and producing a *mux.Router
// as a component for use in dependency injection.
func Server() *S {
	return new(S).
		ServerFactory(ServerConfig{})
}

// ServerFactory sets a custom prototype object that will be unmarshaled
// and used to construct the http.Server and associated Listen strategy.
// By default, ServerConfig{} is used as the factory.  This prototype is
// cloned and unmarshaled using arrange.NewTarget.
//
// The prototype may optionally implement ListenerFactory, which will allow
// custom listen behavior.  If the prototype doesn't implement ListenerFactory,
// then DefaultListenerFactory is used to create the server's net.Listener.
func (s *S) ServerFactory(prototype ServerFactory) *S {
	s.prototype = prototype
	return s
}

// With adds functional options that tailor the *http.Server supplied by
// this builder chain.
func (s *S) With(o ...ServerOption) *S {
	s.options = append(
		s.options,
		ServerOptions(o...).sOption,
	)

	return s
}

// WithRouter adds functional options that tailor the *mux.Router supplied
// by this builder chain.
func (s *S) WithRouter(o ...RouterOption) *S {
	s.options = append(
		s.options,
		RouterOptions(o...).sOption,
	)

	return s
}

// Middleware is a shorthand for a RouterOption that adds several middlewares
// to the *mux.Router being built.
func (s *S) Middleware(m ...func(http.Handler) http.Handler) *S {
	return s.WithRouter(func(router *mux.Router) error {
		for _, f := range m {
			router.Use(f)
		}

		return nil
	})
}

// MiddlewareChain is a shorthand for a RouterOption that adds a chain
// of server middlewares.  Various packages can be used here, such as justinas/alice.
func (s *S) MiddlewareChain(smc ServerMiddlewareChain) *S {
	return s.WithRouter(func(router *mux.Router) error {
		router.Use(smc.Then)
		return nil
	})
}

// ListenerChain adds a ListenerChain that decorates the listener used to accept
// traffic for this server.
func (s *S) ListenerChain(lc ListenerChain) *S {
	s.options = append(
		s.options,
		func(_ *http.Server, _ *mux.Router, chain ListenerChain) (ListenerChain, error) {
			return chain.Extend(lc), nil
		},
	)

	return s
}

// ListenerConstructors adds several decorators for the listener used to accept
// traffic for this server.
func (s *S) ListenerConstructors(l ...ListenerConstructor) *S {
	s.options = append(
		s.options,
		func(_ *http.Server, _ *mux.Router, chain ListenerChain) (ListenerChain, error) {
			return chain.Append(l...), nil
		},
	)

	return s
}

// CaptureListenAddress decorates the server's listener so that the actual address the
// server listens on is sent to a channel when the fx.App is started.
//
// This method is primarily useful during testing or examples when the bind address
// of the server is such that it will bind to an available port, e.g. "", ":0", "[::1]:0", etc.
func (s *S) CaptureListenAddress(ch chan<- net.Addr) *S {
	return s.ListenerConstructors(
		CaptureListenAddress(ch),
	)
}

// Inject allows additional components that tailor an http.Server, mux.Router, or net.Listener.
// These components will be supplied by the enclosing fx.App.
//
// Each value supplied to this method must be a struct value that embeds fx.In.
//
// When the constructor for this server is called, each field of each struct is examined to
// see if it is a type that can apply to tailoring a server, router, or listener.  Any fields
// that cannot be used are silently ignored.
func (s *S) Inject(deps ...interface{}) *S {
	for _, d := range deps {
		if dt, ok := arrange.IsIn(d); ok {
			s.dependencies = append(s.dependencies, dt)
		} else {
			s.errs = append(s.errs,
				fmt.Errorf("%s is not an fx.In struct", dt),
			)
		}
	}

	return s
}

// unmarshalFuncOf determines the function signature for Unmarshal or UnmarshalKey.
// The first input parameter is always a ServerIn struct.  Following that will be any
// fx.In structs, and following that will be any simple dependencies.
func (s *S) unmarshalFuncOf() reflect.Type {
	return reflect.FuncOf(
		// inputs
		append(
			[]reflect.Type{reflect.TypeOf(ServerIn{})},
			s.dependencies...,
		),

		// outputs
		[]reflect.Type{
			reflect.TypeOf((*mux.Router)(nil)),
			arrange.ErrorType(),
		},

		false, // not variadic
	)
}

// unmarshal does all the heavy lifting of unmarshaling a ServerFactory and creating a server, router,
// and binding a listener to the fx.App lifecycle.
//
// If this method does not return an error, it will have bound the listener to the fx.App's Lifecycle.
func (s *S) unmarshal(u func(arrange.Unmarshaler, interface{}) error, inputs []reflect.Value) (router *mux.Router, err error) {
	if len(s.errs) > 0 {
		err = multierr.Combine(s.errs...)
		return
	}

	var (
		target   = arrange.NewTarget(s.prototype)
		serverIn = inputs[0].Interface().(ServerIn)

		p = arrange.NewModulePrinter(Module, serverIn.Printer)
	)

	if err = u(serverIn.Unmarshaler, target.UnmarshalTo.Interface()); err != nil {
		return
	}

	var server *http.Server
	router = mux.NewRouter()
	factory := target.Component.Interface().(ServerFactory)
	if server, err = factory.NewServer(router); err != nil {
		return
	}

	// if the factory did not set a handler, use the router
	if server.Handler == nil {
		server.Handler = router
	}

	var lc ListenerChain
	var optionErrs []error
	for _, dependency := range inputs[1:] {
		arrange.VisitDependencies(
			dependency,
			func(f reflect.StructField, fv reflect.Value) bool {
				if arrange.IsInjected(f, fv) {
					// ignore dependencies that can't be converted
					if so := newSOption(fv.Interface()); so != nil {
						p.Printf("SERVER INJECT => %s.%s %s", dependency.Type(), f.Name, f.Tag)
						if lc, err = so(server, router, lc); err != nil {
							optionErrs = append(optionErrs, err)
						}
					}
				}

				return true
			},
		)
	}

	for _, o := range s.options {
		if lc, err = o(server, router, lc); err != nil {
			optionErrs = append(optionErrs, err)
		}
	}

	if len(optionErrs) > 0 {
		err = multierr.Combine(optionErrs...)
		router = nil
	} else {
		var lf ListenerFactory
		ok := false
		if lf, ok = factory.(ListenerFactory); !ok {
			lf = DefaultListenerFactory{}
		}

		serverIn.Lifecycle.Append(fx.Hook{
			OnStart: ServerOnStart(
				server,
				lc.Factory(lf),
				func() {
					// ensure that if this server exits for any reason,
					// the enclosing fx.App is shutdown
					serverIn.Shutdowner.Shutdown()
				}),
			OnStop: server.Shutdown,
		})
	}

	return
}

// makeUnmarshalFunc dynamically creates the function to be passed as a constructor to the fx.App.
func (s *S) makeUnmarshalFunc(u func(arrange.Unmarshaler, interface{}) error) reflect.Value {
	return reflect.MakeFunc(
		s.unmarshalFuncOf(),
		func(inputs []reflect.Value) []reflect.Value {
			router, err := s.unmarshal(u, inputs)
			return []reflect.Value{
				reflect.ValueOf(router),
				arrange.NewErrorValue(err),
			}
		},
	)
}

// Unmarshal terminates the builder chain and returns a function that produces a mux.Router.
// The *http.Server and net.Listener objects built by this function are not exposed.  However,
// both the server and listener will be bound to the lifecycle of the enclosing fx.App.
func (s *S) Unmarshal() interface{} {
	return s.makeUnmarshalFunc(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.Unmarshal(v)
		},
	).Interface()
}

// UnmarshalKey is like Unmarshal, except that it unmarshals from a particular configuration key.
func (s *S) UnmarshalKey(key string) interface{} {
	return s.makeUnmarshalFunc(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.UnmarshalKey(key, v)
		},
	).Interface()
}

// Provide produces an fx.Provide that does the same thing as Unmarshal.
// is the typical way to leverage this package to create an http.Server.
func (s *S) Provide() fx.Option {
	return fx.Provide(
		s.Unmarshal(),
	)
}

// ProvideKey handles the simple case where a router is built from a given configuration key
// and is exposed as a component of the same name as the key.
func (s *S) ProvideKey(key string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: s.UnmarshalKey(key),
		},
	)
}
