package arrangehttp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func testServerConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverConfig = ServerConfig{
			Address:           ":0",
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 27 * time.Minute,
			WriteTimeout:      38 * time.Second,
			IdleTimeout:       89 * time.Minute,
			MaxHeaderBytes:    478934,
			KeepAlive:         16 * time.Minute,
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, listen, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(listen)
	require.NotNil(server)

	assert.Equal(15*time.Second, server.ReadTimeout)
	assert.Equal(27*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(38*time.Second, server.WriteTimeout)
	assert.Equal(89*time.Minute, server.IdleTimeout)
	assert.Equal(478934, server.MaxHeaderBytes)

	// check that this is a functioning server
	listen = NewListenerChain(CaptureListenAddress(address)).Listen(listen)
	server.Handler = router
	require.NoError(
		ServerOnStart(server, listen)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		conn, err := net.Dial("tcp", listenAddress.String())
		require.NoError(err)
		defer conn.Close()

		fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
		_, err = bufio.NewReader(conn).ReadString('\n')
		require.NoError(err)

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func testServerConfigTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		certificateFile, keyFile = createServerFiles(t)

		serverConfig = ServerConfig{
			Address:           ":0",
			ReadTimeout:       72 * time.Second,
			ReadHeaderTimeout: 109 * time.Minute,
			WriteTimeout:      63 * time.Second,
			IdleTimeout:       9234 * time.Minute,
			MaxHeaderBytes:    3642,
			KeepAlive:         3 * time.Minute,
			TLS: &TLS{
				Certificates: ExternalCertificates{
					{
						CertificateFile: certificateFile,
						KeyFile:         keyFile,
					},
				},
			},
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	defer os.Remove(certificateFile)
	defer os.Remove(keyFile)

	server, listen, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(listen)
	require.NotNil(server)

	assert.Equal(72*time.Second, server.ReadTimeout)
	assert.Equal(109*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(63*time.Second, server.WriteTimeout)
	assert.Equal(9234*time.Minute, server.IdleTimeout)
	assert.Equal(3642, server.MaxHeaderBytes)

	// check that this is a functioning server
	listen = NewListenerChain(CaptureListenAddress(address)).Listen(listen)
	server.Handler = router
	require.NoError(
		ServerOnStart(server, listen)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		conn, err := net.Dial("tcp", listenAddress.String())
		require.NoError(err)
		defer conn.Close()

		fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
		_, err = bufio.NewReader(conn).ReadString('\n')
		require.NoError(err)

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func TestServerConfig(t *testing.T) {
	t.Run("Basic", testServerConfigBasic)
	t.Run("TLS", testServerConfigTLS)
}

func testServerOptionSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		server  = new(http.Server)
		router  = new(mux.Router)
		chain   ListenerChain
	)

	so := ServerOption(func(s *http.Server) error {
		assert.Equal(server, s)
		return nil
	})

	require.NotNil(so)
	c, err := so(server, router, chain)
	assert.Equal(chain, c)
	assert.NoError(err)
}

func testServerOptionFailure(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		server      = new(http.Server)
		router      = new(mux.Router)
		chain       ListenerChain
		expectedErr = errors.New("expected option error")
	)

	so := ServerOption(func(s *http.Server) error {
		assert.Equal(server, s)
		return expectedErr
	})

	require.NotNil(so)
	c, err := so(server, router, chain)
	assert.Equal(chain, c)
	assert.Equal(expectedErr, err)
}

func TestServerOption(t *testing.T) {
	t.Run("Success", testServerOptionSuccess)
	t.Run("Failure", testServerOptionFailure)
}

func testRouterOptionSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		server  = new(http.Server)
		router  = new(mux.Router)
		chain   ListenerChain
	)

	ro := RouterOption(func(r *mux.Router) error {
		assert.Equal(router, r)
		return nil
	})

	require.NotNil(ro)
	c, err := ro(server, router, chain)
	assert.Equal(chain, c)
	assert.NoError(err)
}

func testRouterOptionFailure(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		server      = new(http.Server)
		router      = new(mux.Router)
		chain       ListenerChain
		expectedErr = errors.New("expected option error")
	)

	ro := RouterOption(func(r *mux.Router) error {
		assert.Equal(router, r)
		return expectedErr
	})

	require.NotNil(ro)
	c, err := ro(server, router, chain)
	assert.Equal(chain, c)
	assert.Equal(expectedErr, err)
}

func TestRouterOptions(t *testing.T) {
	t.Run("Success", testRouterOptionSuccess)
	t.Run("Failure", testRouterOptionFailure)
}

func TestMiddleware(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert = assert.New(t)

				m []mux.MiddlewareFunc

				server   = new(http.Server)
				router   = mux.NewRouter()
				chain    ListenerChain
				request  = httptest.NewRequest("GET", "/test", nil)
				response = httptest.NewRecorder()
			)

			for i := 0; i < length; i++ {
				i := i
				m = append(m, func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Header().Set(
							fmt.Sprintf("Decorator-%d", i),
							strconv.Itoa(i),
						)

						next.ServeHTTP(response, request)
					})
				})
			}

			Middleware(m...)(server, router, chain)
			router.HandleFunc("/test", func(response http.ResponseWriter, request *http.Request) {
				response.Header().Set("Called", "true")
			})

			router.ServeHTTP(response, request)

			assert.Equal("true", response.HeaderMap.Get("Called"))
			for i := 0; i < length; i++ {
				assert.Equal(
					strconv.Itoa(i),
					response.HeaderMap.Get(fmt.Sprintf("Decorator-%d", i)),
				)
			}
		})
	}
}

func testServerListenerConstructors(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		address = make(chan net.Addr, 1)

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().
				Use(
					AppendListener(CaptureListenAddress(address)),
				).
				Unmarshal(),
		),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func testServerUnmarshal(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		globalAddress1 = make(chan net.Addr, 1)
		globalAddress2 = make(chan net.Addr, 1)
		localAddress   = make(chan net.Addr, 1)

		globalSOptionCalled = make(chan struct{})

		localSOptionCalled = make(chan struct{})
		localSOption       = ServerOption(func(s *http.Server) error {
			defer close(localSOptionCalled)
			assert.NotNil(s)
			return nil
		})

		v = viper.New()
	)

	type Dependencies struct {
		fx.In
		GlobalSOption             SOption
		GlobalListenerConstructor ListenerConstructor
		GlobalListenerChain       ListenerChain
	}

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			func() ListenerChain {
				return NewListenerChain(
					CaptureListenAddress(globalAddress1),
				)
			},
			func() ListenerConstructor {
				return CaptureListenAddress(globalAddress2)
			},
			func() SOption {
				return ServerOption(func(*http.Server) error {
					close(globalSOptionCalled)
					return nil
				})
			},
			Server(localSOption).
				Inject(Dependencies{}).
				Use(
					ExtendListener(
						NewListenerChain(CaptureListenAddress(localAddress)),
					),
				).
				Unmarshal(),
		),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-localSOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The local server option was not called")
	}

	select {
	case <-globalSOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The global server option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-localAddress:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	select {
	case globalAddress := <-globalAddress1:
		assert.Equal(serverAddress, globalAddress)
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	select {
	case globalAddress := <-globalAddress2:
		assert.Equal(serverAddress, globalAddress)
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

type badServerFactory struct {
	Address string
}

func (bsf badServerFactory) NewServer() (*http.Server, Listen, error) {
	return nil, nil, errors.New("factory error")
}

func testServerServerFactoryError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", "localhost:8080")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().
				ServerFactory(badServerFactory{}).
				Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerLocalSOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", "localhost:8080")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().
				Use(
					ServerOption(func(*http.Server) error { return errors.New("expected server option error") }),
				).
				Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerGlobalSOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	type Dependencies struct {
		fx.In
		Option SOption
	}

	v.Set("address", "localhost:8080")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			func() SOption {
				return ServerOption(func(*http.Server) error {
					return errors.New("expected server option error")
				})
			},
			Server().Inject(Dependencies{}).Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerUnmarshalError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", ":0")
	v.Set("readTimeout", "this is not a valid golang time.Duration")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerUnmarshalBadInject(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			// not a valid fx.In struct
			Server().Inject(123).Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerProvide(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		sOptionCalled = make(chan struct{})
		sOption       = RouterOption(func(r *mux.Router) error {
			defer close(sOptionCalled)
			assert.NotNil(r)
			return nil
		})

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		Server(sOption).
			Use(
				AppendListener(CaptureListenAddress(address)),
			).
			Provide(),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-sOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The server option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func testServerUnmarshalKey(t *testing.T) {
	const yaml = `
servers:
  main:
    address: ":0"
    readTimeout: "30s"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		sOptionCalled = make(chan struct{})
		sOption       = ServerOption(func(s *http.Server) error {
			defer close(sOptionCalled)
			if assert.NotNil(s) {
				assert.Equal(30*time.Second, s.ReadTimeout)
			}

			return nil
		})

		v = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server(sOption).
				Use(
					AppendListener(CaptureListenAddress(address)),
				).
				UnmarshalKey("servers.main"),
		),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-sOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The server option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func testServerUnmarshalKeyError(t *testing.T) {
	const yaml = `
servers:
  main:
    address: ":0"
    readTimeout: "this is not a valid golang time.Duration"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		router  *mux.Router

		v = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().UnmarshalKey("servers.main"),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerUnmarshalKeyBadInject(t *testing.T) {
	const yaml = `
servers:
  main:
    address: ":0"
    readTimeout: "15s"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		router  *mux.Router

		v = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().Inject("this is not an fx.In struct").UnmarshalKey("servers.main"),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerProvideKey(t *testing.T) {
	const yaml = `
servers:
  main:
    address: ":0"
    readTimeout: "30s"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		sOptionCalled = make(chan struct{})
		sOption       = ServerOption(func(s *http.Server) error {
			defer close(sOptionCalled)
			if assert.NotNil(s) {
				assert.Equal(30*time.Second, s.ReadTimeout)
			}

			return nil
		})

		v = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	type RouterIn struct {
		fx.In
		Router *mux.Router `name:"servers.main"`
	}

	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		Server(sOption).
			Use(
				AppendListener(CaptureListenAddress(address)),
			).
			ProvideKey("servers.main"),
		fx.Invoke(
			func(r RouterIn) {
				r.Router.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-sOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The server option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func testServerInject(t *testing.T) {
	type Dependencies struct {
		fx.In
		GlobalOption      SOption
		OptionGroup       []SOption `group:"options"`
		GlobalConstructor ListenerConstructor
		ListenerGroup     []ListenerConstructor `group:"listeners"`
		ListenerChain     ListenerChain
		Middleware        mux.MiddlewareFunc
		MiddlewareGroup   []mux.MiddlewareFunc `group:"middleware"`

		IgnoreMe string `name:"ignore"`
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)

		globalAddress = make(chan net.Addr, 1)
		groupAddress1 = make(chan net.Addr, 1)
		groupAddress2 = make(chan net.Addr, 1)
		chainAddress  = make(chan net.Addr, 1)

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		arrange.Supply(v),
		fx.Provide(
			func() SOption {
				return Middleware(NewHeaders("GlobalOption", "true").AddResponse)
			},
			fx.Annotated{
				Group: "options",
				Target: func() SOption {
					return Middleware(NewHeaders("Option1", "true").AddResponse)
				},
			},
			fx.Annotated{
				Group: "options",
				Target: func() SOption {
					return Middleware(NewHeaders("Option2", "true").AddResponse)
				},
			},
			func() ListenerConstructor {
				return CaptureListenAddress(globalAddress)
			},
			fx.Annotated{
				Group: "listeners",
				Target: func() ListenerConstructor {
					return CaptureListenAddress(groupAddress1)
				},
			},
			fx.Annotated{
				Group: "listeners",
				Target: func() ListenerConstructor {
					return CaptureListenAddress(groupAddress2)
				},
			},
			func() ListenerChain {
				return NewListenerChain(
					CaptureListenAddress(chainAddress),
				)
			},
			func() mux.MiddlewareFunc {
				return NewHeaders("GlobalMiddleware", "true").AddResponse
			},
			fx.Annotated{
				Group: "middleware",
				Target: func() mux.MiddlewareFunc {
					return NewHeaders("Middleware1", "true").AddResponse
				},
			},
			fx.Annotated{
				Group: "middleware",
				Target: func() mux.MiddlewareFunc {
					return NewHeaders("Middleware2", "true").AddResponse
				},
			},
			fx.Annotated{
				Name:   "ignore",
				Target: func() string { return "this should be ignored" },
			},
		),
		Server().Inject(Dependencies{}).Provide(),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	var serverAddress net.Addr
	select {
	case serverAddress = <-globalAddress:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The global option did not execute")
	}

	select {
	case address := <-groupAddress1:
		assert.Equal(serverAddress, address)
	case <-time.After(2 * time.Second):
		assert.Fail("The first group option did not execute")
	}

	select {
	case address := <-groupAddress2:
		assert.Equal(serverAddress, address)
	case <-time.After(2 * time.Second):
		assert.Fail("The second group option did not execute")
	}

	select {
	case address := <-chainAddress:
		assert.Equal(serverAddress, address)
	case <-time.After(2 * time.Second):
		assert.Fail("The listener chain did not execute")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
	assert.Equal(277, response.StatusCode)
	assert.Equal("true", response.Header.Get("GlobalOption"))
	assert.Equal("true", response.Header.Get("Option1"))
	assert.Equal("true", response.Header.Get("Option2"))
	assert.Equal("true", response.Header.Get("GlobalMiddleware"))
	assert.Equal("true", response.Header.Get("Middleware1"))
	assert.Equal("true", response.Header.Get("Middleware2"))

	app.RequireStop()
}

func testServerInjectOptional(t *testing.T) {
	type Dependencies struct {
		fx.In
		GlobalOption        SOption `optional:"true"`
		ListenerConstructor ListenerConstructor
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		arrange.Supply(v),
		fx.Provide(
			func() ListenerConstructor {
				return CaptureListenAddress(address)
			},
		),
		Server().Inject(Dependencies{}).Provide(),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The global option did not execute")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
	assert.Equal(277, response.StatusCode)

	app.RequireStop()
}

func TestServer(t *testing.T) {
	t.Run("ListenerConstructors", testServerListenerConstructors)
	t.Run("Unmarshal", testServerUnmarshal)
	t.Run("UnmarshalError", testServerUnmarshalError)
	t.Run("UnmarshalBadInject", testServerUnmarshalBadInject)
	t.Run("FactoryError", testServerServerFactoryError)
	t.Run("LocalServerOptionError", testServerLocalSOptionError)
	t.Run("GlobalServerOptionError", testServerGlobalSOptionError)
	t.Run("Provide", testServerProvide)
	t.Run("UnmarshalKey", testServerUnmarshalKey)
	t.Run("UnmarshalKeyError", testServerUnmarshalKeyError)
	t.Run("UnmarshalKeyBadInject", testServerUnmarshalKeyBadInject)
	t.Run("ProvideKey", testServerProvideKey)
	t.Run("Inject", testServerInject)
	t.Run("InjectOptional", testServerInjectOptional)
}
