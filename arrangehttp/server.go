package arrangehttp

import (
	"context"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
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

// Server describes how to unmarshal and configure a server, listener,
// and router in the context of an enclosing fx.App.
type Server struct {
	// Name is the optional name of the *mux.Router component
	Name string

	// Key is the configuration key from which this server's factory
	// is unmarshaled.  If Name is not set and this field is set, then
	// this field is used by default as the component name.
	//
	// If this field is unset, unmarshaling takes place at the root
	// of the configuration.
	Key string

	// Unnamed disables the defaulting of a component name when the Key
	// field is set.  Useful when an fx.App only has one server that gets
	// unmarshaled from a key.
	//
	// When this field is true, then the *mux.Router is never named regardless
	// of the other fields.
	Unnamed bool

	// ServerFactory is the prototype instance used to instantiate an *http.Server.
	// If unset, ServerConfig is used.
	//
	// If set, this instance is cloned before unmarshaling.  That means any values
	// set on it will act as defaults.
	ServerFactory ServerFactory

	// Inject is the set of dependencies used to build the server, listener, and router.
	// This is a set of types that are injected when the constructor created by Provide is
	// run.
	//
	// Injected dependencies are always applied before anything in this struct.
	Inject arrange.Inject

	// Options is the set of server options outside the enclosing fx.App that are run
	// before the server is bound to the fx.App lifecycle.  Each element of this sequence
	// must be a function with one of two signatures:
	//
	//   func(*http.Server)
	//   func(*http.Server) error
	Options arrange.Invoke

	// Middleware is the set of decorators for the *mux.Router that come from outside
	// the enclosing fx.App.
	//
	// Any injected middleware, via the Inject field, are applied before anything
	// in this field.
	Middleware alice.Chain

	// ListenerChain is the set of decorators for the net.Listener that come from
	// outside the enclosing fx.App.
	//
	// Any injected listener chains or constructors, via the Inject field, are applied
	// before anything in this field.
	ListenerChain ListenerChain

	// Invoke is the optional set of functions executed as an fx.Invoke option.  These functions
	// are executed after server and listener construction.  Each element of this sequence
	// must be a function with one of two signatures:
	//
	//   func(*mux.Router)
	//   func(*mux.Router) error
	//
	// If this slice is empty, client code must add at least one fx.Invoke that accepts the
	// *mux.Router or else the server created by this struct will not get started.
	Invoke arrange.Invoke
}

func (s *Server) name() string {
	switch {
	case s.Unnamed:
		return ""
	case len(s.Name) > 0:
		return s.Name
	default:
		return s.Key
	}
}

func (s *Server) unmarshal(u arrange.Unmarshaler) (sf ServerFactory, err error) {
	prototype := s.ServerFactory
	if prototype == nil {
		prototype = ServerConfig{}
	}

	target := arrange.NewTarget(prototype)
	sf = target.Component.Interface().(ServerFactory)
	if len(s.Key) > 0 {
		err = u.UnmarshalKey(s.Key, target.UnmarshalTo.Interface())
	} else {
		err = u.Unmarshal(target.UnmarshalTo.Interface())
	}

	return
}

func (s *Server) configure(in ServerIn, server *http.Server, deps []reflect.Value) (lc ListenerChain, err error) {
	var (
		middleware alice.Chain
		options    arrange.Invoke
	)

	arrange.VisitDependencies(
		func(d arrange.Dependency) bool {
			if d.Injected() {
				arrange.TryConvert(
					d.Value.Interface(),
					func(v alice.Chain) {
						middleware = middleware.Extend(v)
					},
					func(v alice.Constructor) {
						middleware = middleware.Append(v)
					},
					func(v []alice.Constructor) {
						middleware = middleware.Append(v...)
					},
					func(v ListenerChain) {
						lc = lc.Extend(v)
					},
					func(v ListenerConstructor) {
						lc = lc.Append(v)
					},
					func(v []ListenerConstructor) {
						lc = lc.Append(v...)
					},
					func(v func(*http.Server)) {
						options = append(options, v)
					},
					func(v []func(*http.Server)) {
						for _, o := range v {
							options = append(options, o)
						}
					},
					func(v func(*http.Server) error) {
						options = append(options, v)
					},
					func(v []func(*http.Server) error) {
						for _, o := range v {
							options = append(options, o)
						}
					},
				)
			}

			return true
		},
		deps...,
	)

	middleware = middleware.Extend(s.Middleware)
	lc = lc.Extend(s.ListenerChain)
	err = multierr.Append(
		err,
		options.Apply(server),
	)

	if err == nil {
		server.Handler = middleware.Then(server.Handler)
	}

	return
}

// provide implements the main workflow.  It's a Template Method that unmarshals
// and creates an *http.Server, configures it, and binds a net.Listener to the fx.App lifecycle.
func (s *Server) provide(deps []reflect.Value) (router *mux.Router, err error) {
	// the first dependency is always a ServerIn
	in := deps[0].Interface().(ServerIn)

	var sf ServerFactory
	sf, err = s.unmarshal(in.Unmarshaler)
	if err != nil {
		return
	}

	router = mux.NewRouter()
	var server *http.Server
	server, err = sf.NewServer(router)
	if err != nil {
		return
	}

	var lc ListenerChain
	lc, err = s.configure(in, server, deps[1:])
	if err != nil {
		return
	}

	var lf ListenerFactory
	if v, ok := sf.(ListenerFactory); ok {
		lf = v
	} else {
		lf = DefaultListenerFactory{}
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: ServerOnStart(
			server,
			lc.Factory(lf),
			ShutdownOnExit(in.Shutdowner),
		),
		OnStop: server.Shutdown,
	})

	return
}

// Provide creates an fx.Option that bootstraps an HTTP server.  An *mux.Router
// component is returned to the enclosing fx.App, but the server and listener are
// not exposed as components.
//
// The constructor supplied to the enclosing fx.App always has a ServerIn as an
// input parameter followed by each type contained in the Inject field (if any).
// This dynamically created constructor implements a basic workflow:
//
//   - A clone of the ServerFactory object is unmarshaled.  An instance of ServerConfig
//     is used if no ServerFactory is supplied.
//
//   - The ServerFactory is passed a new *mux.Router (as an http.Handler) to instantiate
//     the *http.Server object.
//
//   - Each injected value, dictated by the types in Inject, are examined to see if they
//     contain dependencies that apply to building a server (see below).  Those dependencies
//     are applied to the server or to a ListenerChain which will eventually decorate the net.Listener.
//
//   - Each functional option in the Inject dependencies or Options is executed with the server instance
//
//   - Any middleware found in the Inject dependencies or Middleware are applied to the *mux.Router
//
//   - Assuming no errors thus far, an fx.Lifecycle hook is created that binds the *http.Server
//     object to the fx.App lifecycle.  The OnStart function in this hook creates a net.Listener
//     and decorates that listener with any application dependencies.
//
//   - If Invoke is not empty, then an fx.Invoke option is also created that is injected with
//     the *mux.Router instance created above and executes each Invoke closure.
//
// The set of dependencies in Inject that can apply to an *http.Server are very flexible:
//
//   - anything convertible to an alice.Constructor or alice.Chain will decorate the *mux.Router.
//     This also includes slices of alice.Constructor, which is just a func(http.Handler) http.Handler.
//
//   - anything convertible to a ListenerConstructor or ListenerChain will decorate the net.Listener
//     Included in this are slices of ListenerConstructor.
//
//   - any function type that takes a sole parameter of *http.Server and returns either nothing
//     or an error will be executed as a server option along with everything in the Options field.
//     This also includes slices of the same function types.
func (s *Server) Provide() fx.Option {
	provideFunc := arrange.Inject{reflect.TypeOf(ServerIn{})}.
		Extend(s.Inject).
		MakeFunc(s.provide)

	name := s.name()
	var options []fx.Option
	if len(name) > 0 {
		options = append(options, fx.Provide(
			fx.Annotated{
				Name:   name,
				Target: provideFunc.Interface(),
			},
		))
	} else {
		options = append(options, fx.Provide(
			provideFunc.Interface(),
		))
	}

	if len(s.Invoke) > 0 {
		var invokeFunc reflect.Value
		if len(name) > 0 {
			// build an fx.In struct
			invokeFunc = arrange.Inject{
				arrange.Struct{}.In().Append(
					arrange.Field{
						Name: name,
						Type: (*mux.Router)(nil),
					},
				),
			}.MakeFunc(
				func(inputs []reflect.Value) error {
					// the router will always be the 2nd field of the only struct parameter
					router := inputs[0].Field(1).Interface().(*mux.Router)
					return s.Invoke.Apply(router)
				},
			)
		} else {
			// just a simple global, unnamed dependency
			invokeFunc = arrange.Inject{
				(*mux.Router)(nil),
			}.MakeFunc(
				func(inputs []reflect.Value) error {
					return s.Invoke.Apply(inputs[0])
				},
			)
		}

		options = append(options, fx.Invoke(
			invokeFunc.Interface(),
		))
	}

	return fx.Options(options...)
}
