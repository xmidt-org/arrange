package arrangehttp

import (
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
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

// ServerOption is a functional option that is allowed to mutate an http.Server
// prior to binding it to an uber/fx App.  Server options can supply application
// logic that doesn't come from external configuration:
//
//   v := viper.New()
//   fx.New(
//     arrange.Supply(v),
//     arrangehttp.Server(func(s *http.Server) error {
//       s.ConnState = func(c net.Conn, cs http.ConnState) {
//         // custom application connection state handling, e.g. logging
//       }
//
//       return nil
//     }).Provide(),
//     fx.Provide(
//       func(r *mux.Router) MyComponent {
//         // although the http.Server is not a component, this mux.Router
//         // will be the handler for the server with a custom ConnState
//       },
//     ),
//   )
type ServerOption func(*http.Server) error

// RouterOption is a functional option that can mutate a mux.Router prior to
// it being returned as a component in an uber/fx App.  Router options can
// supply custom application tailoring to a router that doesn't come
// from external configuration:
//
//   v := viper.New()
//   fx.New(
//     arrange.Supply(v),
//     arrangehttp.Server().
//       RouterOptions(func(r *mux.Router) error {
//         r.StrictSlash(true)
//         r.Use(myGlobalMiddleware)
//         return nil
//     }).Provide(),
//     fx.Provide(
//       func(r *mux.Router) MyComponent {
//         // this router will have some global middleware and strict slash turned on
//       },
//     ),
//   )
type RouterOption func(*mux.Router) error

// Middleware creates a RouterOption that applies the given decorators to
// the mux.Router.  Multiple Middleware options are cumulative.
func Middleware(m ...func(http.Handler) http.Handler) RouterOption {
	return func(r *mux.Router) error {
		// have to do a for loop here to get around some golang type madness
		for _, f := range m {
			r.Use(f)
		}

		return nil
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

	// ListenerChain is an optional component that, if supplied, will apply
	// to all servers.
	ListenerChain ListenerChain `optional:"true"`

	// ServerOptions will apply to all http.Servers, if supplied
	ServerOptions []ServerOption `optional:"true"`

	// RouterOptions will apply to all mux.Routers, if supplied
	RouterOptions []RouterOption `optional:"true"`
}

// S is a Fluent Builder for unmarshaling an http.Server.  This type must be
// created with the Server function.
type S struct {
	so        []ServerOption
	ro        []RouterOption
	chain     ListenerChain
	prototype ServerFactory
}

// Server starts a Fluent Builder method chain for creating an http.Server,
// binding its lifecycle to the fx.App lifecycle, and producing a *mux.Router
// as a component for use in dependency injection.
func Server(opts ...ServerOption) *S {
	s := new(S)
	if len(opts) > 0 {
		// safe copy
		s.so = append([]ServerOption{}, opts...)
	}

	return s.ServerFactory(ServerConfig{})
}

// RouterOptions supplies options that can modify the *mux.Router prior to
// it being returned as a component.  The set of router options is appended
// with each call to this method.
func (s *S) RouterOptions(opts ...RouterOption) *S {
	s.ro = append(s.ro, opts...)
	return s
}

// Use adds ListenerConstructors that will decorate the server's net.Listener
// as part of server startup
func (s *S) Use(more ...ListenerConstructor) *S {
	s.chain = s.chain.Append(more...)
	return s
}

// UseChain adds an entire chain of constructors that will decorate the server's
// net.Listener as part of server startup
func (s *S) UseChain(more ListenerChain) *S {
	s.chain = s.chain.Extend(more)
	return s
}

// ServerFactory sets the prototype object, as described by arrange.NewTarget,
// that will be unmarshaled and used to instantiate the *http.Server and listener.
// By default, ServerConfig is used.  This method can be used to override the
// factory with custom configuration.
func (s *S) ServerFactory(prototype ServerFactory) *S {
	s.prototype = prototype
	return s
}

// newRouter does all the the work of creating the server, binding its lifecycle
// to the fx.App, and setting up the *mux.Router.
func (s *S) newRouter(f ServerFactory, in ServerIn) (*mux.Router, error) {
	server, listen, err := f.NewServer()
	if err != nil {
		return nil, err
	}

	for _, f := range in.ServerOptions {
		if err := f(server); err != nil {
			return nil, err
		}
	}

	for _, f := range s.so {
		if err := f(server); err != nil {
			return nil, err
		}
	}

	router := mux.NewRouter()
	for _, f := range in.RouterOptions {
		if err := f(router); err != nil {
			return nil, err
		}
	}

	for _, f := range s.ro {
		if err := f(router); err != nil {
			return nil, err
		}
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: ServerOnStart(
			server,
			in.ListenerChain.Extend(s.chain).Listen(listen),
			ShutdownOnExit(in.Shutdowner),
		),
		OnStop: server.Shutdown,
	})

	server.Handler = router
	return router, nil
}

// Unmarshal uses an injected Viper instance to unmarshal the ServerFactory, which
// is then used to create the *http.Server and listener as well as binding the server's
// lifecycle to the fx.App.
//
// This method terminates the builder chain, and must be used inside fx.Provide:
//
//   v := viper.New() // setup not shown
//   fx.New(
//     arrange.Supply(v), // don't forget to supply the viper as a component!
//     fx.Provide(
//       arrangehttp.Server().Unmarshal(),
//     ),
//     fx.Invoke(
//       func(r *mux.Router) error {
//         // add any routes or other modifications to the router,
//         // which will be the handler for the server
//       },
//     ),
//   )
//
// Generally, Provide is preferred over Unmarshal.  However, Unmarshal allows one
// to name the router component or to place it into a group:
//
//   v := viper.New()
//
//   type RouterIn struct {
//     fx.In
//     Router *mux.Router `name:"myServer"`
//   }
//
//   fx.New(
//     arrange.Supply(v),
//     fx.Provide(
//       fx.Annotated{
//         Name: "myServer",
//         Target: arrangehttp.Server().Unmarshal(),
//       },
//     ),
//     fx.Invoke(
//       func(r RouterIn) error {
//         // r.Router will hold the router used as the handler for the server
//       },
//     ),
//   )
func (s *S) Unmarshal(opts ...viper.DecoderConfigOption) func(ServerIn) (*mux.Router, error) {
	return func(in ServerIn) (*mux.Router, error) {
		var (
			target = arrange.NewTarget(s.prototype)
			err    = in.Viper.Unmarshal(
				target.UnmarshalTo(),
				arrange.Merge(in.DecoderOptions, opts),
			)
		)

		if err != nil {
			return nil, err
		}

		return s.newRouter(
			target.Component().(ServerFactory),
			in,
		)
	}
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

// UnmarshalKey is similar to Unmarshal, but unmarshals a particular Viper configuration
// key rather than unmarshaling from the root.
//
// Assume a yaml configuration similar to:
//
//   servers:
//     main:
//       address: ":8080"
//       readTimeout: "60s"
//
// The corresponding UnmarshalKey declaration would be:
//
//   v := viper.New() // read in the above YAML
//   fx.New(
//     arrange.Supply(v), // don't forget to supply the viper as a component!
//     fx.Provide(
//       arrangehttp.Server().UnmarshalKey("servers.main"),
//     ),
//     fx.Invoke(
//       func(r *mux.Router) error {
//         // this router is the server's handler
//       },
//     ),
//   )
//
// Note that UnmarshalKey simply provides a constructor, as with Unmarshal.  To name
// the component, one has to use fx.Annotated.  ProvideKey does this automatically.
func (s *S) UnmarshalKey(key string, opts ...viper.DecoderConfigOption) func(ServerIn) (*mux.Router, error) {
	return func(in ServerIn) (*mux.Router, error) {
		var (
			target = arrange.NewTarget(s.prototype)
			err    = in.Viper.UnmarshalKey(
				key,
				target.UnmarshalTo(),
				arrange.Merge(in.DecoderOptions, opts),
			)
		)

		if err != nil {
			return nil, err
		}

		return s.newRouter(
			target.Component().(ServerFactory),
			in,
		)
	}
}

// ProvideKey unmarshals the ServerFactory from a particular Viper key.  The *mux.Router
// component is named the same as that key.
//
//   v := viper.New()
//
//   type RouterIn struct {
//     fx.In
//     Router *mux.Router `name:"servers.main"` // note that this name is the same as the key
//   }
//
//   fx.New(
//     arrange.Supply(v),
//     arrangehttp.Server().ProvideKey("servers.main"),
//     fx.Invoke(
//       func(r RouterIn) error {
//         // r.Router will hold the router used as the handler for the server
//       },
//     ),
//   )
func (s *S) ProvideKey(key string, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: s.UnmarshalKey(key, opts...),
		},
	)
}
