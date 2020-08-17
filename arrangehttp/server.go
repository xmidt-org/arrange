package arrangehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
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
	// information was unmarshaled into this instance.
	NewServer() (*http.Server, error)
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
	TLS               *arrangetls.Config
}

// NewServer is the built-in implementation of ServerFactory in this package.
// This should serve most needs.  Nothing needs to be done to use this implementation.
// By default, a Fluent Builder chain begun with Server() will use ServerConfig.
func (sc ServerConfig) NewServer() (server *http.Server, err error) {
	server = &http.Server{
		Addr:              sc.Address,
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

// SOption is a functional option used to tailor an http.Server and its dependent
// objects.  SOptions are evaluated at construction time but before the http.Server
// is bound to the fx.App lifecycle.
type SOption func(*http.Server, *mux.Router, ListenerChain) (ListenerChain, error)

// SOptions aggregates several SOption instances into a single option
func SOptions(options ...SOption) SOption {
	return func(s *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
		var err error
		for _, so := range options {
			c, err = so(s, r, c)
			if err != nil {
				break
			}
		}

		return c, err
	}
}

// NewSOption reflects an object and tries to convert it into an SOption.  The set
// of types allowed is flexible:
//
//   (1) SOption or a slice of same
//   (2) ServerOption or a closure that accepts an *http.Server and may return an error
//   (3) RouterOption or a closure that accepts an *mux.Router and may return an error
//   (4) ListenerConstructor or a slice of same
//   (5) ListenerChain
//   (6) mux.MiddlewareFunc, a slice of same, or a closure of the same signature as mux.MiddlewareFunc
//
// Any other type will produce an error.
func NewSOption(o interface{}) (so SOption, err error) {
	switch o := o.(type) {
	case SOption:
		so = o

	case []SOption:
		so = SOptions(o...)

	// this really isn't necessary, but it's consistent with NewCOption
	case func(*http.Server, *mux.Router, ListenerChain) (ListenerChain, error):
		so = o

	case ServerOption:
		so = func(s *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c, o(s)
		}

	case func(*http.Server) error:
		so = func(s *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c, o(s)
		}

	case func(*http.Server):
		so = func(s *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			o(s)
			return c, nil
		}

	case RouterOption:
		so = func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c, o(r)
		}

	case func(*mux.Router) error:
		so = func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c, o(r)
		}

	case func(*mux.Router):
		so = func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			o(r)
			return c, nil
		}

	case ListenerConstructor:
		so = func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c.Append(o), nil
		}

	case []ListenerConstructor:
		so = func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c.Append(o...), nil
		}

	case ListenerChain:
		so = func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c.Extend(o), nil
		}

	case mux.MiddlewareFunc:
		so = func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			r.Use(o)
			return c, nil
		}

	case []mux.MiddlewareFunc:
		so = func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			r.Use(o...)
			return c, nil
		}

	case func(http.Handler) http.Handler:
		so = func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			r.Use(o)
			return c, nil
		}

	default:
		err = fmt.Errorf("%s is not supported as an SOption", reflect.TypeOf(o))
	}

	return
}

// ServerOption is a functional option type that can be converted to an SOption.
// This type exists primarily to make fx.Provide constructors more concise.
type ServerOption func(*http.Server) error

// RouterOption is a functional option type that can be converted to an SOption.
// This type exists primarily to make fx.Provide constructors more concise.
type RouterOption func(*mux.Router) error

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
func Server(o ...interface{}) *S {
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

// Use applies options to this builder.  The set of types allowed are any
// of the types that can be supplied to NewSOption as well as instances
// of structs embedded with fx.In.
//
// Anything convertible to an SOption is evaluated at construction time.
//
// Any fx.In struct is used as an injectible set of dependencies.  Fields on
// that struct are converted into SOptions using the same rules as NewSOption,
// but any struct field not convertible is ignored.
func (s *S) Use(v ...interface{}) *S {
	for _, o := range v {
		so, err := NewSOption(o)
		if err == nil {
			s.options = append(s.options, so)
			continue
		}

		if dependency, ok := arrange.IsIn(o); ok {
			s.dependencies = append(s.dependencies, dependency.Type())
			continue
		}

		s.errs = append(s.errs,
			err,
			fmt.Errorf("%s does not refer to an fx.In struct", reflect.TypeOf(v)),
		)
	}

	return s
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
	var options []SOption

	// visit struct fields in dependencies, building SOptions where possible
	for _, d := range dependencies {
		arrange.VisitFields(
			d,
			func(f reflect.StructField, fv reflect.Value) arrange.VisitResult {
				if arrange.IsDependency(f, fv) {
					// ignore struct fields that aren't applicable
					// this allows callers to reuse fx.In structs for different purposes
					if so, err := NewSOption(fv.Interface()); err == nil {
						options = append(options, so)
					}
				}

				return arrange.VisitContinue
			},
		)
	}

	// locally defined options execute after injected options, allowing
	// local options to override global ones
	options = append(options, s.options...)
	for _, so := range options {
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
