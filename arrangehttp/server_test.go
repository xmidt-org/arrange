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
	"github.com/xmidt-org/arrange/arrangetls"
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
			TLS: &arrangetls.Config{
				Certificates: arrangetls.ExternalCertificates{
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

func testSOptionsSuccess(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert    = assert.New(t)
				require   = require.New(t)
				server    = new(http.Server)
				router    = mux.NewRouter()
				chain     ListenerChain
				options   []SOption
				callCount int
			)

			for i := 0; i < length; i++ {
				options = append(options, func(s *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
					assert.Equal(server, s)
					assert.Equal(router, r)
					assert.Equal(chain, c)
					callCount++
					return c, nil
				})
			}

			so := SOptions(options...)
			require.NotNil(so)
			c, err := so(server, router, chain)
			assert.NoError(err)
			assert.Equal(chain, c)
			assert.Equal(length, callCount)
		})
	}
}

func testSOptionsFailure(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		server      = new(http.Server)
		router      = mux.NewRouter()
		chain       ListenerChain
		expectedErr = errors.New("expected option error")
		so          = SOptions(
			func(s *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
				assert.Equal(server, s)
				assert.Equal(router, r)
				assert.Equal(chain, c)
				return c, nil
			},
			func(s *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
				assert.Equal(server, s)
				assert.Equal(router, r)
				assert.Equal(chain, c)
				return c, expectedErr
			},
			func(s *http.Server, r *mux.Router, c ListenerChain) (ListenerChain, error) {
				assert.Fail("This option should not have been called")
				return c, errors.New("This option should not have been called")
			},
		)
	)

	require.NotNil(so)
	c, err := so(server, router, chain)
	assert.Equal(expectedErr, err)
	assert.Equal(chain, c)
}

func TestSOptions(t *testing.T) {
	t.Run("Success", testSOptionsSuccess)
	t.Run("Failure", testSOptionsFailure)
}

func testNewSOptionUnsupported(t *testing.T) {
	assert := assert.New(t)
	so, err := NewSOption("this is not supported as an SOption")
	assert.Error(err)
	assert.Nil(so)
}

func testNewSOptionSimple(t *testing.T) {
	var (
		assert          = assert.New(t)
		require         = require.New(t)
		called          = false
		option  SOption = func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			called = true
			return c, nil
		}
		so, err = NewSOption(option)
	)

	require.NoError(err)
	require.NotNil(so)
	_, err = so(nil, nil, NewListenerChain())
	assert.NoError(err)
	assert.True(called)
}

func testNewSOptionClosure(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		called  = false
		option  = func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
			called = true
			return c, nil
		}
		so, err = NewSOption(option)
	)

	require.NoError(err)
	require.NotNil(so)
	_, err = so(nil, nil, NewListenerChain())
	assert.NoError(err)
	assert.True(called)
}

func testNewSOptionComposite(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		called0 = false
		called1 = false
		options = []SOption{
			func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
				called0 = true
				return c, nil
			},
			func(_ *http.Server, _ *mux.Router, c ListenerChain) (ListenerChain, error) {
				called1 = true
				return c, nil
			},
		}

		so, err = NewSOption(options)
	)

	require.NoError(err)
	require.NotNil(so)
	_, err = so(nil, nil, NewListenerChain())
	assert.NoError(err)
	assert.True(called0)
	assert.True(called1)
}

func testNewSOptionServer(t *testing.T) {
	var (
		actualServer = new(*http.Server)
		optionErr    = errors.New("expected option error")
		testData     = []struct {
			option      interface{}
			expectedErr error
		}{
			{
				option: ServerOption(func(s *http.Server) error {
					*actualServer = s
					return nil
				}),
			},
			{
				option: ServerOption(func(s *http.Server) error {
					*actualServer = s
					return optionErr
				}),
				expectedErr: optionErr,
			},
			{
				option: func(s *http.Server) error {
					*actualServer = s
					return nil
				},
			},
			{
				option: func(s *http.Server) error {
					*actualServer = s
					return optionErr
				},
				expectedErr: optionErr,
			},
			{
				option: func(s *http.Server) {
					*actualServer = s
				},
			},
		}
	)

	for i, record := range testData {
		*actualServer = nil
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				server  = new(http.Server)
				chain   ListenerChain
				so, err = NewSOption(record.option)
			)

			require.NoError(err)
			require.NotNil(so)
			c, err := so(server, nil, chain)
			assert.Equal(chain, c)
			assert.Equal(record.expectedErr, err)
			assert.Equal(server, *actualServer)
		})
	}
}

func testNewSOptionRouter(t *testing.T) {
	var (
		actualRouter = new(*mux.Router)
		optionErr    = errors.New("expected option error")
		testData     = []struct {
			option      interface{}
			expectedErr error
		}{
			{
				option: RouterOption(func(r *mux.Router) error {
					*actualRouter = r
					return nil
				}),
			},
			{
				option: RouterOption(func(r *mux.Router) error {
					*actualRouter = r
					return optionErr
				}),
				expectedErr: optionErr,
			},
			{
				option: func(r *mux.Router) error {
					*actualRouter = r
					return nil
				},
			},
			{
				option: func(r *mux.Router) error {
					*actualRouter = r
					return optionErr
				},
				expectedErr: optionErr,
			},
			{
				option: func(r *mux.Router) {
					*actualRouter = r
				},
			},
		}
	)

	for i, record := range testData {
		*actualRouter = nil
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				router  = mux.NewRouter()
				chain   ListenerChain
				so, err = NewSOption(record.option)
			)

			require.NoError(err)
			require.NotNil(so)
			c, err := so(nil, router, chain)
			assert.Equal(chain, c)
			assert.Equal(record.expectedErr, err)
			assert.Equal(router, *actualRouter)
		})
	}
}

func testNewSOptionListener(t *testing.T) {
	var (
		address0                     = make(chan net.Addr, 1)
		lc0      ListenerConstructor = CaptureListenAddress(address0)

		address1                     = make(chan net.Addr, 1)
		lc1      ListenerConstructor = CaptureListenAddress(address1)

		address2                     = make(chan net.Addr, 1)
		lc2      ListenerConstructor = CaptureListenAddress(address2)

		testData = []struct {
			option interface{}
			ch     []<-chan net.Addr
		}{
			{
				option: lc1,
				ch:     []<-chan net.Addr{address1},
			},
			{
				option: []ListenerConstructor{lc1, lc2},
				ch:     []<-chan net.Addr{address1, address2},
			},
			{
				option: NewListenerChain(lc1, lc2),
				ch:     []<-chan net.Addr{address1, address2},
			},
		}
	)

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				server  = new(http.Server)
				router  = mux.NewRouter()
				chain   = NewListenerChain(lc0)
				so, err = NewSOption(record.option)
			)

			require.NoError(err)
			require.NotNil(so)

			c, err := so(server, router, chain)
			require.NoError(err)

			l, err := net.Listen("tcp", ":0")
			require.NoError(err)
			defer l.Close()

			// the act of invoking the decorator will capture addresses
			c.Then(l)

			var expected net.Addr
			select {
			case expected = <-address0:
				// passing
			case <-time.After(time.Second):
				assert.Fail("The initial chain was not used")
			}

			for _, ch := range record.ch {
				select {
				case actual := <-ch:
					assert.Equal(expected, actual)
				case <-time.After(time.Second):
					assert.Fail("Decorator options were not used")
				}
			}
		})
	}
}

func testNewSOptionMiddleware(t *testing.T) {
	testData := []struct {
		option   interface{}
		expected http.Header
	}{
		{
			option: mux.MiddlewareFunc(NewHeaders("Option", "true").AddResponse),
			expected: http.Header{
				"Option": {"true"},
			},
		},
		{
			option: []mux.MiddlewareFunc{
				NewHeaders("Option1", "true").AddResponse,
				NewHeaders("Option2", "true").AddResponse,
			},
			expected: http.Header{
				"Option1": {"true"},
				"Option2": {"true"},
			},
		},
		{
			option: NewHeaders("Option", "true").AddResponse,
			expected: http.Header{
				"Option": {"true"},
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				server  = new(http.Server)
				router  = mux.NewRouter()
				chain   ListenerChain
				so, err = NewSOption(record.option)
			)

			require.NoError(err)
			require.NotNil(so)

			_, err = so(server, router, chain)
			require.NoError(err)

			router.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(234)
			})

			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/test", nil)
			router.ServeHTTP(response, request)

			assert.Equal(record.expected, response.HeaderMap)
			assert.Equal(234, response.Code)
		})
	}
}

func TestNewSOption(t *testing.T) {
	t.Run("Unsupported", testNewSOptionUnsupported)
	t.Run("Simple", testNewSOptionSimple)
	t.Run("Closure", testNewSOptionClosure)
	t.Run("Composite", testNewSOptionComposite)
	t.Run("Server", testNewSOptionServer)
	t.Run("Router", testNewSOptionRouter)
	t.Run("Listener", testNewSOptionListener)
	t.Run("Middleware", testNewSOptionMiddleware)
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
					CaptureListenAddress(address),
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
		GlobalSOption             ServerOption
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
			func() ServerOption {
				return func(*http.Server) error {
					close(globalSOptionCalled)
					return nil
				}
			},
			Server(localSOption).
				Use(
					Dependencies{},
					NewListenerChain(CaptureListenAddress(localAddress)),
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
			func() ServerOption {
				return func(*http.Server) error {
					return errors.New("expected server option error")
				}
			},
			Server().Use(Dependencies{}).Unmarshal(),
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

func testServerUnmarshalUseError(t *testing.T) {
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
			// not a valid option
			Server().Use(123).Unmarshal(),
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
				CaptureListenAddress(address),
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
					CaptureListenAddress(address),
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
			Server().Use("this is not an fx.In struct").UnmarshalKey("servers.main"),
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
				CaptureListenAddress(address),
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

func TestServer(t *testing.T) {
	t.Run("ListenerConstructors", testServerListenerConstructors)
	t.Run("Unmarshal", testServerUnmarshal)
	t.Run("UnmarshalError", testServerUnmarshalError)
	t.Run("UnmarshalUseError", testServerUnmarshalUseError)
	t.Run("FactoryError", testServerServerFactoryError)
	t.Run("LocalServerOptionError", testServerLocalSOptionError)
	t.Run("GlobalServerOptionError", testServerGlobalSOptionError)
	t.Run("Provide", testServerProvide)
	t.Run("UnmarshalKey", testServerUnmarshalKey)
	t.Run("UnmarshalKeyError", testServerUnmarshalKeyError)
	t.Run("UnmarshalKeyBadInject", testServerUnmarshalKeyBadInject)
	t.Run("ProvideKey", testServerProvideKey)
}
