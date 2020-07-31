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

type ServerFactory interface {
	NewServer() (*http.Server, Listen, error)
}

type ServerConfig struct {
	Network           string
	Address           string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	KeepAlive         time.Duration
	Tls               *ServerTls
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

	server.TLSConfig, err = NewServerTlsConfig(sc.Tls)
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

type ServerOption func(*http.Server) error

type RouterOption func(*mux.Router) error

type ServerIn struct {
	arrange.ProvideIn

	Lifecycle  fx.Lifecycle
	Shutdowner fx.Shutdowner
}

type S struct {
	so        []ServerOption
	ro        []RouterOption
	prototype ServerFactory
}

func Server(opts ...ServerOption) *S {
	return &S{
		so: opts,
	}
}

func (s *S) ServerFactory(prototype ServerFactory) *S {
	s.prototype = prototype
	return s
}

func (s *S) RouterOptions(opts ...RouterOption) *S {
	s.ro = append(s.ro, opts...)
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
		OnStart: ServerOnStart(server, listen, ShutdownOnExit(in.Shutdowner)),
		OnStop:  server.Shutdown,
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
