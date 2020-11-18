package arrangehttp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type simpleServerFactory struct {
	Address   string
	returnErr error
}

func (ssf simpleServerFactory) NewServer(h http.Handler) (*http.Server, error) {
	if ssf.returnErr != nil {
		return nil, ssf.returnErr
	}

	return &http.Server{
		Addr:    ssf.Address,
		Handler: h,
	}, nil
}

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

	server, err := serverConfig.NewServer(router)
	require.NoError(err)
	require.NotNil(server)
	assert.Equal(router, server.Handler)

	assert.Equal(15*time.Second, server.ReadTimeout)
	assert.Equal(27*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(38*time.Second, server.WriteTimeout)
	assert.Equal(89*time.Minute, server.IdleTimeout)
	assert.Equal(478934, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
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
						CertificateFile: CertificateFile,
						KeyFile:         KeyFile,
					},
				},
			},
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, err := serverConfig.NewServer(router)
	require.NoError(err)
	require.NotNil(server)
	assert.Equal(router, server.Handler)

	assert.Equal(72*time.Second, server.ReadTimeout)
	assert.Equal(109*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(63*time.Second, server.WriteTimeout)
	assert.Equal(9234*time.Minute, server.IdleTimeout)
	assert.Equal(3642, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
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

func testServerConfigHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverConfig = ServerConfig{
			Header: http.Header{
				"test1": {"true"},
				"test2": {"1", "2"},
			},
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, err := serverConfig.NewServer(router)
	require.NoError(err)
	require.NotNil(server)
	assert.NotNil(server.Handler)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		response, err := http.Get("http://" + listenAddress.String())
		require.NoError(err)
		require.NotNil(response)
		defer response.Body.Close()
		io.Copy(ioutil.Discard, response.Body)

		assert.Equal([]string{"true"}, response.Header["Test1"])
		assert.Equal([]string{"1", "2"}, response.Header["Test2"])

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func TestServerConfig(t *testing.T) {
	t.Run("Basic", testServerConfigBasic)
	t.Run("TLS", testServerConfigTLS)
	t.Run("Header", testServerConfigHeader)
}

type ServerTestSuite struct {
	suite.Suite
	testLogger fx.Option

	viper       *viper.Viper
	serverAddr  chan net.Addr
	captureAddr ListenerConstructor
}

func (suite *ServerTestSuite) SetupTest() {
	suite.testLogger = arrange.TestLogger(suite.T())
	suite.viper = viper.New()

	suite.serverAddr = make(chan net.Addr, 1)
	suite.captureAddr = CaptureListenAddress(suite.serverAddr)
}

func (suite *ServerTestSuite) handler(response http.ResponseWriter, _ *http.Request) {
	// write an odd response code to easily verify that this handler executed
	response.WriteHeader(299)
}

func (suite *ServerTestSuite) configureRoutes(r *mux.Router) {
	r.HandleFunc("/test", suite.handler)
}

func (suite *ServerTestSuite) yaml(v string) {
	suite.viper.SetConfigType("yaml")

	suite.Require().NoError(
		suite.viper.ReadConfig(strings.NewReader(v)),
	)
}

func (suite *ServerTestSuite) requireServerAddr() net.Addr {
	a, _ := AwaitListenAddress(
		suite.Require().FailNow,
		suite.serverAddr,
		5*time.Second,
	)

	return a
}

func (suite *ServerTestSuite) serverURL() string {
	return "http://" + suite.requireServerAddr().String() + "/test"
}

func (suite *ServerTestSuite) checkServer() *http.Response {
	response, err := http.Get(suite.serverURL())
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()

	suite.Equal(299, response.StatusCode)
	return response
}

func (suite *ServerTestSuite) TestUnmarshalError() {
	suite.yaml(`
readTimeout: "EXPECTED ERROR: this is not a valid duration"
`)

	app := fx.New(
		suite.testLogger,
		arrange.ForViper(suite.viper),
		Server{
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Error(app.Err())
}

func (suite *ServerTestSuite) TestServerFactoryError() {
	app := fx.New(
		suite.testLogger,
		arrange.ForViper(suite.viper),
		Server{
			ServerFactory: simpleServerFactory{
				returnErr: errors.New("expected"),
			},
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Error(app.Err())
}

func (suite *ServerTestSuite) TestConfigureError() {
	app := fx.New(
		suite.testLogger,
		arrange.ForViper(suite.viper),
		Server{
			Options: arrange.Invoke{
				func(s *http.Server) error {
					suite.NotNil(s)
					return errors.New("expected")
				},
			},
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Error(app.Err())
}

func (suite *ServerTestSuite) TestDefaults() {
	app := fxtest.New(
		suite.T(),
		suite.testLogger,
		arrange.ForViper(suite.viper),
		Server{
			ListenerChain: NewListenerChain(
				suite.captureAddr,
			),
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	suite.checkServer()
	app.RequireStop()
}

func (suite *ServerTestSuite) TestUnnamed() {
	suite.yaml(`
servers:
  main:
    address: ":0"
`)

	app := fxtest.New(
		suite.T(),
		suite.testLogger,
		arrange.ForViper(suite.viper),
		Server{
			Key:     "servers.main",
			Unnamed: true,
			ListenerChain: NewListenerChain(
				suite.captureAddr,
			),
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	suite.checkServer()
	app.RequireStop()
}

func (suite *ServerTestSuite) TestNamed() {
	suite.yaml(`
servers:
  main:
    address: ":0"
`)

	app := fxtest.New(
		suite.T(),
		suite.testLogger,
		arrange.ForViper(suite.viper),
		Server{
			Name: "foobar",
			Key:  "servers.main",
			ListenerChain: NewListenerChain(
				suite.captureAddr,
			),
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	suite.checkServer()
	app.RequireStop()
}

func (suite *ServerTestSuite) TestDefaultListenerFactory() {
	app := fxtest.New(
		suite.T(),
		suite.testLogger,
		arrange.ForViper(suite.viper),
		Server{
			ServerFactory: simpleServerFactory{}, // this doesn't implement ListenerFactory
			ListenerChain: NewListenerChain(
				suite.captureAddr,
			),
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	suite.checkServer()
	app.RequireStop()
}

func (suite *ServerTestSuite) TestMiddleware() {
	suite.yaml(`
servers:
  main:
    address: ":0"
`)

	app := fxtest.New(
		suite.T(),
		suite.testLogger,
		arrange.ForViper(suite.viper),
		fx.Provide(
			func() func(http.Handler) http.Handler {
				return func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Header().Set("Injected-Unnamed-Constructor", "true")
						next.ServeHTTP(response, request)
					})
				}
			},
			func() alice.Chain {
				return alice.New(
					func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							response.Header().Set("Injected-Unnamed-Chain", "true")
							next.ServeHTTP(response, request)
						})
					},
				)
			},
			fx.Annotated{
				Name: "constructor",
				Target: func() alice.Constructor {
					return func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							response.Header().Set("Injected-Named-Constructor", "true")
							next.ServeHTTP(response, request)
						})
					}
				},
			},
			fx.Annotated{
				Group: "constructors",
				Target: func() func(http.Handler) http.Handler {
					return func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							response.Header().Add("Injected-Constructor-Group", "1")
							next.ServeHTTP(response, request)
						})
					}
				},
			},
			fx.Annotated{
				Group: "constructors",
				Target: func() func(http.Handler) http.Handler {
					return func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
							response.Header().Add("Injected-Constructor-Group", "2")
							next.ServeHTTP(response, request)
						})
					}
				},
			},
		),
		Server{
			Key: "servers.main",
			Inject: arrange.Inject{
				func(http.Handler) http.Handler { return nil },
				alice.Chain{},
				arrange.Struct{}.In().Append(
					arrange.Field{
						Name: "constructor",
						Type: alice.Constructor(func(http.Handler) http.Handler { return nil }),
					},
					arrange.Field{
						Group: "constructors",
						Type:  func(http.Handler) http.Handler { return nil },
					},
				).Of(),
			},
			ListenerChain: NewListenerChain(
				suite.captureAddr,
			),
			Middleware: alice.New(
				func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Header().Set("External-Constructor", "true")
						next.ServeHTTP(response, request)
					})
				},
			),
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	response := suite.checkServer()
	suite.Equal(
		"true",
		response.Header.Get("External-Constructor"),
	)

	suite.Equal(
		"true",
		response.Header.Get("Injected-Unnamed-Constructor"),
	)

	suite.Equal(
		"true",
		response.Header.Get("Injected-Unnamed-Chain"),
	)

	suite.Equal(
		"true",
		response.Header.Get("Injected-Named-Constructor"),
	)

	suite.ElementsMatch(
		[]string{"1", "2"},
		response.Header.Values("Injected-Constructor-Group"),
	)

	app.RequireStop()
}

func (suite *ServerTestSuite) TestListener() {
	suite.yaml(`
servers:
  main:
    address: ":0"
`)

	var called []string

	app := fxtest.New(
		suite.T(),
		suite.testLogger,
		arrange.ForViper(suite.viper),
		fx.Provide(
			func() ListenerConstructor {
				return func(next net.Listener) net.Listener {
					called = append(called, "injected-unnamed-constructor")
					return next
				}
			},
			fx.Annotated{
				Name: "constructor",
				Target: func() ListenerConstructor {
					return func(next net.Listener) net.Listener {
						called = append(called, "injected-named-constructor")
						return next
					}
				},
			},
			func() ListenerChain {
				return NewListenerChain(
					func(next net.Listener) net.Listener {
						called = append(called, "injected-unnamed-chain")
						return next
					},
				)
			},
			fx.Annotated{
				Group: "constructors",
				Target: func() ListenerConstructor {
					return func(next net.Listener) net.Listener {
						called = append(called, "injected-constructor-group-1")
						return next
					}
				},
			},
			fx.Annotated{
				Group: "constructors",
				Target: func() ListenerConstructor {
					return func(next net.Listener) net.Listener {
						called = append(called, "injected-constructor-group-2")
						return next
					}
				},
			},
		),
		Server{
			Inject: arrange.Inject{
				struct {
					fx.In
					F1 ListenerConstructor
					F2 ListenerConstructor `name:"constructor"`
					F3 ListenerChain
					F4 []ListenerConstructor `group:"constructors"`
				}{},
			},
			ListenerChain: NewListenerChain(
				suite.captureAddr,
				func(next net.Listener) net.Listener {
					called = append(called, "external")
					return next
				},
			),
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	suite.checkServer()
	suite.ElementsMatch(
		[]string{
			"injected-unnamed-constructor",
			"injected-named-constructor",
			"injected-unnamed-chain",
			"injected-constructor-group-1",
			"injected-constructor-group-2",
			"external",
		},
		called,
	)

	app.RequireStop()
}

func (suite *ServerTestSuite) TestOptions() {
	suite.yaml(`
servers:
  main:
    address: ":0"
`)

	var called []string

	app := fxtest.New(
		suite.T(),
		suite.testLogger,
		arrange.ForViper(suite.viper),
		fx.Provide(
			func() func(*http.Server) {
				return func(s *http.Server) {
					suite.NotNil(s)
					called = append(called, "injected")
				}
			},
			func() func(*http.Server) error {
				return func(s *http.Server) error {
					suite.NotNil(s)
					called = append(called, "injected-with-error")
					return nil
				}
			},
			fx.Annotated{
				Group: "options",
				Target: func() func(*http.Server) {
					return func(s *http.Server) {
						suite.NotNil(s)
						called = append(called, "group-1")
					}
				},
			},
			fx.Annotated{
				Group: "options",
				Target: func() func(*http.Server) {
					return func(s *http.Server) {
						suite.NotNil(s)
						called = append(called, "group-2")
					}
				},
			},
			fx.Annotated{
				Group: "options-with-error",
				Target: func() func(*http.Server) error {
					return func(s *http.Server) error {
						suite.NotNil(s)
						called = append(called, "group-with-error-1")
						return nil
					}
				},
			},
			fx.Annotated{
				Group: "options-with-error",
				Target: func() func(*http.Server) error {
					return func(s *http.Server) error {
						suite.NotNil(s)
						called = append(called, "group-with-error-2")
						return nil
					}
				},
			},
		),
		Server{
			Inject: arrange.Inject{
				struct {
					fx.In
					F1 func(*http.Server)
					F2 func(*http.Server) error
					F3 []func(*http.Server)       `group:"options"`
					F4 []func(*http.Server) error `group:"options-with-error"`
				}{},
			},
			ListenerChain: NewListenerChain(
				suite.captureAddr,
			),
			Options: arrange.Invoke{
				func(s *http.Server) {
					suite.NotNil(s)
					called = append(called, "external")
				},
				func(s *http.Server) error {
					suite.NotNil(s)
					called = append(called, "external-with-error")
					return nil
				},
			},
			Invoke: arrange.Invoke{
				suite.configureRoutes,
			},
		}.Provide(),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()
	defer app.Stop(context.Background())

	suite.checkServer()
	suite.ElementsMatch(
		[]string{
			"injected",
			"injected-with-error",
			"group-1",
			"group-2",
			"group-with-error-1",
			"group-with-error-2",
			"external",
			"external-with-error",
		},
		called,
	)

	app.RequireStop()
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
