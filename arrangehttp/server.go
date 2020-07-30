package arrangehttp

import (
	"context"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
)

var routerType reflect.Type = reflect.TypeOf((*mux.Router)(nil))

func RouterType() reflect.Type {
	return routerType
}

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

func (sc ServerConfig) NewServer() (server *http.Server, listen Listen, err error) {
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
		listen = ListenerFactory{
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

func (s *S) newRouter(factory ServerFactory, shutdowner fx.Shutdowner) (router *mux.Router, hook fx.Hook, err error) {
	var server *http.Server
	var listen Listen
	server, listen, err = factory.NewServer()
	if err != nil {
		return
	}

	for i := 0; i < len(s.so); i++ {
		if err = s.so[i](server); err != nil {
			return
		}
	}

	router = mux.NewRouter()
	for i := 0; i < len(s.ro); i++ {
		if err = s.ro[i](router); err != nil {
			return
		}
	}

	server.Handler = router
	hook.OnStop = server.Shutdown
	hook.OnStart = func(ctx context.Context) error {
		listener, err := listen(ctx, server)
		if err != nil {
			return err
		}

		go func() {
			defer shutdowner.Shutdown()
			server.Serve(listener)
		}()

		return nil
	}

	return
}

func (s *S) Unmarshal(opts ...viper.DecoderConfigOption) interface{} {
	target := s.newTarget()
	return reflect.MakeFunc(
		reflect.FuncOf(
			// inputs:
			[]reflect.Type{reflect.TypeOf(ServerIn{})},

			// outputs:
			[]reflect.Type{RouterType(), arrange.ErrorType()},

			// not variadic:
			false,
		),
		func(args []reflect.Value) []reflect.Value {
			var (
				in  = args[0].Interface().(ServerIn)
				err = in.Viper.Unmarshal(
					target.UnmarshalTo(),
					arrange.Merge(in.DecodeOptions, opts),
				)

				router     *mux.Router
				serverHook fx.Hook
			)

			if err == nil {
				router, serverHook, err = s.newRouter(
					target.Component().(ServerFactory),
					in.Shutdowner,
				)
			}

			if err == nil {
				in.Lifecycle.Append(serverHook)
			}

			return []reflect.Value{
				reflect.ValueOf(router),
				arrange.NewErrorValue(err),
			}
		},
	).Interface()
}

func (s *S) Provide(opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		s.Unmarshal(opts...),
	)
}

func (s *S) UnmarshalKey(key string, opts ...viper.DecoderConfigOption) interface{} {
	target := s.newTarget()
	return reflect.MakeFunc(
		reflect.FuncOf(
			// inputs:
			[]reflect.Type{reflect.TypeOf(ServerIn{})},

			// outputs:
			[]reflect.Type{RouterType(), arrange.ErrorType()},

			// not variadic:
			false,
		),
		func(args []reflect.Value) []reflect.Value {
			var (
				in  = args[0].Interface().(ServerIn)
				err = in.Viper.UnmarshalKey(
					key,
					target.UnmarshalTo(),
					arrange.Merge(in.DecodeOptions, opts),
				)

				router     *mux.Router
				serverHook fx.Hook
			)

			if err == nil {
				router, serverHook, err = s.newRouter(
					target.Component().(ServerFactory),
					in.Shutdowner,
				)
			}

			if err == nil {
				in.Lifecycle.Append(serverHook)
			}

			return []reflect.Value{
				reflect.ValueOf(router),
				arrange.NewErrorValue(err),
			}
		},
	).Interface()
}

func (s *S) ProvideKey(key string, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: s.UnmarshalKey(key, opts...),
		},
	)
}
