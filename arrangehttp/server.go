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
	if len(options) == 1 {
		return options[0]
	}

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
//   (1) SOption or any type convertible to an SOption
//   (2) ServerOption or any type convertible to a ServerOption
//   (3) Any type convertible to a func(*http.Server), which is basically a ServerOption that returns no error
//   (4) RouterOption or any type convertible to a RouterOption
//   (5) Any type convertible to a func(*mux.Router), which is basically a RouterOption that returns no error
//   (6) ListenerConstructor or any type convertible to a ListenerConstructor
//   (7) mux.MiddlewareFunc or any type convertible to a mux.MiddlewareFunc (including an alice.Constructor)
//   (8) ListenerChain
//   (9) Any slice or array of the above, which are applied in the slice element order
//
// Any other type will produce an error.
func NewSOption(o interface{}) (SOption, error) {
	v := reflect.ValueOf(o)

	// handled types noted below:

	// SOption
	// []SOption
	if o, ok := tryConvertToOptionSlice(v, SOption(nil)); ok {
		return SOptions(o.([]SOption)...), nil
	}

	// func(http.Handler) http.Handler
	// []func(http.Handler) http.Handler
	// mux.MiddlewareFunc
	// []mux.MiddlewareFunc
	// alice.Constructor
	// []alice.Constructor
	if m, ok := tryConvertToOptionSlice(v, mux.MiddlewareFunc(nil)); ok {
		return func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			r.Use(m.([]mux.MiddlewareFunc)...)
			return c, nil
		}, nil
	}

	// func(net.Listener) net.Listener
	// []func(net.Listener) net.Listener
	// ListenerConstructor
	// []ListenerConstructor
	if lc, ok := tryConvertToOptionSlice(v, ListenerConstructor(nil)); ok {
		return func(s *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			return c.Append(lc.([]ListenerConstructor)...), nil
		}, nil
	}

	// ServerOption
	// []ServerOption
	// func(*http.Server) error
	// []func(*http.Server) error
	if o, ok := tryConvertToOptionSlice(v, ServerOption(nil)); ok {
		return func(s *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			for _, f := range o.([]ServerOption) {
				if err := f(s); err != nil {
					return c, err
				}
			}

			return c, nil
		}, nil
	}

	// explicitly support a ServerOption variant that returns no error
	// this helps reduce code noise when there are lots of options,
	// avoiding "return nil" all over the place
	if o, ok := tryConvertToOptionSlice(v, (func(*http.Server))(nil)); ok {
		return func(s *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			for _, f := range o.([]func(*http.Server)) {
				f(s)
			}

			return c, nil
		}, nil
	}

	// RouterOption
	// []RouterOption
	// func(*mux.Router) error
	// []func(*mux.Router) error
	if o, ok := tryConvertToOptionSlice(v, RouterOption(nil)); ok {
		return func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			for _, f := range o.([]RouterOption) {
				if err := f(r); err != nil {
					return c, err
				}
			}

			return c, nil
		}, nil
	}

	// explicitly support a RouterOption variant that returns no error
	// this helps reduce code noise when there are lots of options,
	// avoiding "return nil" all over the place
	if o, ok := tryConvertToOptionSlice(v, (func(*mux.Router))(nil)); ok {
		return func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			for _, f := range o.([]func(*mux.Router)) {
				f(r)
			}

			return c, nil
		}, nil
	}

	// ListenerChain
	if o, ok := tryConvertToOptionSlice(v, ListenerChain{}); ok {
		return func(_ *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
			for _, lc := range o.([]ListenerChain) {
				c = c.Extend(lc)
			}

			return c, nil
		}, nil
	}

	return nil, fmt.Errorf("%s is not supported as an SOption", reflect.TypeOf(o))
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

// unmarshalOptions returns a slice of SOptions (possibly) created from both this builder's context
// and the supplied dependencies, if any.  if the supplied dependencies slice is empty, this
// method simply returns s.options.
func (s *S) unmarshalOptions(p fx.Printer, dependencies []reflect.Value) (options []SOption) {
	if len(dependencies) > 0 {
		// visit struct fields in dependencies, building SOptions where possible
		for _, d := range dependencies {
			arrange.VisitFields(
				d,
				func(f reflect.StructField, fv reflect.Value) arrange.VisitResult {
					if arrange.IsDependency(f, fv) {
						// ignore struct fields that aren't applicable
						// this allows callers to reuse fx.In structs for different purposes
						raw := fv.Interface()
						if so, err := NewSOption(raw); err == nil {
							p.Printf("SERVER OPTION => %T %s", raw, f.Tag)
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
	} else {
		// optimization: for the case with no dependencies, don't bother
		// making a copy and just return the builder's options as is
		options = s.options
	}

	return
}

// applyUnmarshal does all the heavy-lifting of creating an http.Server and mux.Router and
// applying any options.  If everything is successful, the http.Server is bound to the
// fx.Lifecycle.  The returned function will always return the tuple of (*mux.Router, error),
// and the first input parameter will always be a ServerIn.
func (s *S) applyUnmarshal(uf func(arrange.Unmarshaler, interface{}) error) interface{} {
	return reflect.MakeFunc(
		reflect.FuncOf(
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
		),
		func(inputs []reflect.Value) []reflect.Value {
			var (
				router *mux.Router
				err    error
			)

			if len(s.errs) > 0 {
				err = multierr.Combine(s.errs...)
			} else {
				var (
					server  *http.Server
					chain   ListenerChain
					factory ServerFactory
				)

				target := arrange.NewTarget(s.prototype)
				in := inputs[0].Interface().(ServerIn)
				err = uf(in.Unmarshaler, target.UnmarshalTo.Interface())
				if err == nil {
					factory = target.Component.Interface().(ServerFactory)
					server, err = factory.NewServer()
				}

				if err == nil {
					router = mux.NewRouter()
					server.Handler = router
					for _, so := range s.unmarshalOptions(in.Printer, inputs[1:]) {
						chain, err = so(server, router, chain)
						if err != nil {
							break
						}
					}
				}

				if err == nil {
					lf, ok := factory.(ListenerFactory)
					if !ok {
						lf = DefaultListenerFactory{}
					}

					// if everything's good, bind the server to the fx.App lifecycle
					in.Lifecycle.Append(fx.Hook{
						OnStart: ServerOnStart(
							server,
							chain.Factory(lf),
							ShutdownOnExit(in.Shutdowner),
						),
						OnStop: server.Shutdown,
					})
				}
			}

			return []reflect.Value{
				reflect.ValueOf(router),
				arrange.NewErrorValue(err),
			}
		},
	).Interface()
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
func (s *S) Unmarshal() interface{} {
	return s.applyUnmarshal(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.Unmarshal(v)
		},
	)
}

func (s *S) UnmarshalKey(key string) interface{} {
	return s.applyUnmarshal(
		func(u arrange.Unmarshaler, v interface{}) error {
			return u.UnmarshalKey(key, v)
		},
	)
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
func (s *S) Provide() fx.Option {
	return fx.Provide(
		s.Unmarshal(),
	)
}

func (s *S) ProvideKey(key string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: s.UnmarshalKey(key),
		},
	)
}
