package arrangehttp

import (
	"fmt"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// ServerFactory is the creation strategy for both an http.Server and the
// particular listener used for the accept loop.  This interface is implemented
// by any unmarshaled struct which hold server configuration fields.
type ServerFactory interface {
	// NewServer is responsible for creating an http.Server using whatever
	// information was unmarshaled into this instance.
	//
	// The Listen strategy is used to create the net.Listener for the server's
	// accept loop.  Since various parts of this listener can be driven by configuration,
	// for example the connection keep alive, this method must supply a non-nil Listen.
	// See ListenerFactory for a convenient way to provide a Listen closure.
	NewServer() (*http.Server, Listen, error)
}

// ServerConfig is the built-in ServerFactory implementation for this package.
// This struct can be unmarshaled via Viper, thus allowing an http.Server to
// be bootstrapped from external configuration.
type ServerConfig struct {
	Network           string
	Address           string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	KeepAlive         time.Duration
	TLS               *TLS
}

// NewServer is the built-in implementation of ServerFactory in this package.
// This should serve most needs.  Nothing needs to be done to use this implementation.
// By default, a Fluent Builder chain begun with Server() will use ServerConfig.
func (sc ServerConfig) NewServer() (server *http.Server, l Listen, err error) {
	server = &http.Server{
		Addr:              sc.Address,
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		WriteTimeout:      sc.WriteTimeout,
		IdleTimeout:       sc.IdleTimeout,
		MaxHeaderBytes:    sc.MaxHeaderBytes,
	}

	server.TLSConfig, err = NewTLSConfig(sc.TLS)
	if err == nil {
		l = ListenerFactory{
			ListenConfig: net.ListenConfig{
				KeepAlive: sc.KeepAlive,
			},
			Network: sc.Network,
		}.Listen
	}

	return
}

// SOption is a functional option used to tailor an http.Server and its dependent
// objects.  SOptions are evaluated at construction time but before the http.Server
// is bound to the fx.App lifecycle.
type SOption func(*http.Server, *mux.Router, ListenerChain) (ListenerChain, error)

// ServerOption is a convenience for an SOption that just modifies the http.Server
func ServerOption(f func(*http.Server) error) SOption {
	return func(s *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
		return c, f(s)
	}
}

// RouterOption is a convenience for an SOption that just modifies the mux.Router
func RouterOption(f func(*mux.Router) error) SOption {
	return func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
		return c, f(r)
	}
}

// Middleware applies middleware to the mux.Router.  Note that when injecting
// options from dependencies, you can also supply mux.MiddlewareFunc components
// directly.
func Middleware(m ...mux.MiddlewareFunc) SOption {
	return RouterOption(func(r *mux.Router) error {
		r.Use(m...)
		return nil
	})
}

// AppendListener produces an SOption that adds ListenerConstructors that
// will participate in decorating the server's net.Listener.
func AppendListener(more ...ListenerConstructor) SOption {
	return func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
		return c.Append(more...), nil
	}
}

// ExtendListener produces an SOption that adds another ListenerChain that
// will participate in decorating the server's net.Listener.
func ExtendListener(more ListenerChain) SOption {
	return func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
		return c.Extend(more), nil
	}
}

// ServerIn describes the set of dependencies for creating a mux.Router and,
// by extension, an http.Server.
type ServerIn struct {
	arrange.ProvideIn

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
	options      []SOption
	dependencies []reflect.Type
	prototype    ServerFactory
}

// Server starts a Fluent Builder method chain for creating an http.Server,
// binding its lifecycle to the fx.App lifecycle, and producing a *mux.Router
// as a component for use in dependency injection.
func Server(o ...SOption) *S {
	return new(S).
		ServerFactory(ServerConfig{}).
		Use(o...)
}

// ServerFactory sets a custom prototype object that will be unmarshaled
// and used to construct the http.Server and associated Listen strategy.
// By default, ServerConfig{} is used as the factory.
func (s *S) ServerFactory(prototype ServerFactory) *S {
	s.prototype = prototype
	return s
}

// Use applies options to this builder.  These options will be evaluated
// at construction time but before a server is bound to the fx.App lifecycle.
func (s *S) Use(o ...SOption) *S {
	s.options = append(s.options, o...)
	return s
}

// Inject applies dependencies from the surrounding fx.App to Unmarshal, UnmarshalKey,
// Provide, or ProvideKey.  Each of the values supplied to this method must be a struct value
// that embeds fx.In or a pointer to same.  When constructors created by this builder are
// invoked, each of the struct fields are examined to see if they are options that this
// builder can apply.  Other fields are ignored.
//
// The available options that can appear as dependency fields in structs are:
//
//   - SOption
//   - []SOption
//   - ListenerConstructor
//   - []ListenerConstructor
//   - ListenerChain
//   - mux.MiddlewareFunc
//   - []mux.MiddlewareFunc
//
// The fields of each dependency struct are applied in the order they are declared.
// Thus, Inject preserves the order of things like mux.MiddlewareFuncs.
//
//   // MyDependencies fields will be applied in this declared order,
//   // regardless of the order they appear in fx.New()
//   type MyDependencies struct {
//     fx.In // required!
//     Logging     arrangehttp.SOption `name:"logging"`
//     RateLimiter arrangehttp.SOption `name:"rateLimiter"`
//     Security    arrangehttp.SOption `name:"security"`
//   }
//
//   v := viper.New()
//   fx.New(
//     arrange.Supply(v),
//     fx.Provide(
//       fx.Annotated{
//         Name: "rateLimiter",
//         Target: func() arrangehttp.SOption {
//           return arrangehttp.RouterOption(
//             NewRateLimiterMiddleware(),
//           )
//         },
//       },
//       fx.Annotated{
//         Name: "security",
//         Target: func() arrangehttp.SOption {
//           return arrangehttp.RouterOption(
//             NewSecurityMiddleware(),
//           )
//         },
//       },
//       fx.Annotated{
//         Name: "logging",
//         Target: func() arrangehttp.SOption {
//           return arrangehttp.RouterOption(
//             NewLoggingMiddleware(),
//           )
//         },
//       },
//     ),
//     // this could also be Unmarshal, UnmarshalKey, or ProvideKey
//     arrangehttp.Server().Inject(MyDependencies{}).Provide(),
//   )
func (s *S) Inject(values ...interface{}) *S {
	for _, v := range values {
		if dependency, ok := arrange.IsIn(v); ok {
			s.dependencies = append(s.dependencies, dependency.Type())
		} else {
			// use the original type, since IsIn will often return a different type
			s.errs = append(s.errs, fmt.Errorf("%s does not refer to a struct", reflect.TypeOf(v)))
		}
	}

	return s
}

// applyServerDependency applies the value of an fx.In struct field
func applyServerDependency(s *http.Server, r *mux.Router, c ListenerChain, d interface{}) (ListenerChain, error) {
	var err error
	switch d := d.(type) {
	case SOption:
		c, err = d(s, r, c)

	// this allows `group:"..."` injection
	case []SOption:
		for _, so := range d {
			c, err = so(s, r, c)
		}

	case ListenerConstructor:
		c = c.Append(d)

	// this allows `group:"..."` injection
	case []ListenerConstructor:
		c = c.Append(d...)

	case ListenerChain:
		c = c.Extend(d)

	case mux.MiddlewareFunc:
		r.Use(d)

	// this allows `group:"..."` injection
	case []mux.MiddlewareFunc:
		r.Use(d...)
	}

	return c, err
}

// newRouter does all the heavy-lifting of creating an http.Server and mux.Router and
// applying any options.  If everything is successful, the http.Server is bound to the
// fx.Lifecycle.
func (s *S) newRouter(f ServerFactory, in ServerIn, dependencies []reflect.Value) (*mux.Router, error) {
	server, listen, err := f.NewServer()
	if err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	server.Handler = router
	var chain ListenerChain

	// first: apply any injected dependencies
	for _, d := range dependencies {
		var err error
		arrange.VisitFields(
			d,
			func(f reflect.StructField, fv reflect.Value) arrange.VisitResult {
				if !arrange.IsDependency(f, fv) {
					return arrange.VisitContinue
				}

				chain, err = applyServerDependency(server, router, chain, fv.Interface())
				if err != nil {
					return arrange.VisitTerminate
				} else {
					return arrange.VisitContinue
				}
			},
		)

		if err != nil {
			return nil, err
		}
	}

	// second: apply any locally defined options
	for _, so := range s.options {
		chain, err = so(server, router, chain)
		if err != nil {
			return nil, err
		}
	}

	// if everything's good, bind the server to the fx.App lifecycle
	in.Lifecycle.Append(fx.Hook{
		OnStart: ServerOnStart(
			server,
			chain.Listen(listen),
			ShutdownOnExit(in.Shutdowner),
		),
		OnStop: server.Shutdown,
	})

	return router, nil
}

// unmarshalFuncOf returns the function signature for an unmarshal function.
// The first parameter will always be a ServerIn.  If more than one parameter
// is supplied, they will all be structs expected to be injected by uber/fx.
// The return values are always (*mux.Router, error).
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

// Unmarshal terminates the builder chain and returns a function that produces a mux.Router.
// The returned function will accept the ServerIn dependency struct along with any structs
// supplied via Inject.  The returned mux.Router will be the handler of a server bound to
// the fx.App lifecycle.
//
//   v := viper.New()
//   fx.New(
//     arrange.Supply(v),
//     fx.Provide(
//       func() http.Handler { /* create a handler */ },
//       Server().Unmarshal(),
//     ),
//     fx.Invoke(
//       func(r *mux.Router, h http.Handler) {
//         // This router is the handler for the above server.
//         r.Handle("/", h)
//       },
//     ),
//   )
func (s *S) Unmarshal(opts ...viper.DecoderConfigOption) interface{} {
	return reflect.MakeFunc(
		s.unmarshalFuncOf(),
		func(inputs []reflect.Value) []reflect.Value {
			var router *mux.Router
			var err error

			if len(s.errs) > 0 {
				err = multierr.Combine(s.errs...)
			} else {
				in := inputs[0].Interface().(ServerIn)
				target := arrange.NewTarget(s.prototype)
				err = in.Viper.Unmarshal(
					target.UnmarshalTo(),
					arrange.Merge(in.DecoderOptions, opts),
				)

				if err == nil {
					router, err = s.newRouter(
						target.Component().(ServerFactory),
						in,
						inputs[1:],
					)
				}
			}

			return []reflect.Value{
				reflect.ValueOf(router),
				arrange.NewErrorValue(err),
			}
		},
	).Interface()
}

func (s *S) UnmarshalKey(key string, opts ...viper.DecoderConfigOption) interface{} {
	return reflect.MakeFunc(
		s.unmarshalFuncOf(),
		func(inputs []reflect.Value) []reflect.Value {
			var router *mux.Router
			var err error

			if len(s.errs) > 0 {
				err = multierr.Combine(s.errs...)
			} else {
				in := inputs[0].Interface().(ServerIn)
				target := arrange.NewTarget(s.prototype)
				err = in.Viper.UnmarshalKey(
					key,
					target.UnmarshalTo(),
					arrange.Merge(in.DecoderOptions, opts),
				)

				if err == nil {
					router, err = s.newRouter(
						target.Component().(ServerFactory),
						in,
						inputs[1:],
					)
				}
			}

			return []reflect.Value{
				reflect.ValueOf(router),
				arrange.NewErrorValue(err),
			}
		},
	).Interface()
}

// Provide produces an fx.Provide that does the same thing as Unmarshal.  This
// is the typical way to leverage this package to create an http.Server:
//
//   v := viper.New() // setup not shown
//   fx.New(
//     arrange.Supply(v), // don't forget to supply the viper as a component!
//     arrangehttp.Server().Provide(),
//     fx.Invoke(
//       func(r *mux.Router) error {
//         // add any routes or other modifications to the router,
//         // which will be the handler for the server
//       },
//     ),
//   )
//
// Use Unmarshal instead of this method when more control over the created component
// is necessary, such as putting it in a group or naming it.
func (s *S) Provide(opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		s.Unmarshal(opts...),
	)
}

func (s *S) ProvideKey(key string, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: s.UnmarshalKey(key, opts...),
		},
	)
}
