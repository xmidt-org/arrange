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
	TLS               *ServerTLS
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

	server.TLSConfig, err = NewServerTLSConfig(sc.TLS)
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
// prior to binding it to an uber/fx App
type ServerOption func(*http.Server) error

// RouterOption is a functional option that can mutate a mux.Router prior to
// it being returned as a component in an uber/fx App.  A RouterOption does not
// participate in dependency injection; rather, its purpose is to establish
// custom settings that do not depend on components, such as any global middleware
// or settings such as SkipClean.
type RouterOption func(*mux.Router) error

// ServerIn describes the set of dependencies for creating a mux.Router and,
// by extension, an http.Server.
type ServerIn struct {
	arrange.ProvideIn

	Lifecycle  fx.Lifecycle
	Shutdowner fx.Shutdowner
}

// S is a Fluent Builder for unmarshaling an http.Server.  This type is typically
// created with the Server function.
type S struct {
	so        []ServerOption
	ro        []RouterOption
	chain     ListenerChain
	prototype ServerFactory
}

func Server(opts ...ServerOption) *S {
	return &S{
		so: opts,
	}
}

func (s *S) RouterOptions(opts ...RouterOption) *S {
	s.ro = append(s.ro, opts...)
	return s
}

func (s *S) AppendListenerConstructors(more ...ListenerConstructor) *S {
	s.chain = s.chain.Append(more...)
	return s
}

func (s *S) ExtendListenerConstructors(more ListenerChain) *S {
	s.chain = s.chain.Extend(more)
	return s
}

func (s *S) ServerFactory(prototype ServerFactory) *S {
	s.prototype = prototype
	return s
}

func (s *S) newTarget() arrange.Target {
	prototype := s.prototype
	if prototype == nil {
		prototype = ServerConfig{}
	}

	return arrange.NewTarget(prototype)
}

func (s *S) newRouter(f ServerFactory, in ServerIn) (*mux.Router, error) {
	server, listen, err := f.NewServer()
	if err != nil {
		return nil, err
	}

	for _, f := range s.so {
		if err := f(server); err != nil {
			return nil, err
		}
	}

	router := mux.NewRouter()
	for _, f := range s.ro {
		if err := f(router); err != nil {
			return nil, err
		}
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: ServerOnStart(
			server,
			s.chain.Listen(listen),
			ShutdownOnExit(in.Shutdowner),
		),
		OnStop: server.Shutdown,
	})

	return router, nil
}

func (s *S) Unmarshal(opts ...viper.DecoderConfigOption) func(ServerIn) (*mux.Router, error) {
	return func(in ServerIn) (*mux.Router, error) {
		var (
			target = s.newTarget()
			err    = in.Viper.Unmarshal(
				target.UnmarshalTo(),
				arrange.Merge(in.DecodeOptions, opts),
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

func (s *S) Provide(opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		s.Unmarshal(opts...),
	)
}

func (s *S) UnmarshalKey(key string, opts ...viper.DecoderConfigOption) func(ServerIn) (*mux.Router, error) {
	return func(in ServerIn) (*mux.Router, error) {
		var (
			target = s.newTarget()
			err    = in.Viper.UnmarshalKey(
				key,
				target.UnmarshalTo(),
				arrange.Merge(in.DecodeOptions, opts),
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

func (s *S) ProvideKey(key string, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: s.UnmarshalKey(key, opts...),
		},
	)
}
