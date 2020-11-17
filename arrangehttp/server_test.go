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

func (ssf simpleServerFactory) NewServer(http.Handler) (*http.Server, error) {
	if ssf.returnErr != nil {
		return nil, ssf.returnErr
	}

	return &http.Server{
		Addr: ssf.Address,
		// this factory does not set a handler, forcing the infrastructure to set it
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

func (suite *ServerTestSuite) serverGet() *http.Response {
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
readTimeout: "this is not a valid duration"
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

	defer app.Stop(context.Background())
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

	defer app.Stop(context.Background())
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

	suite.serverGet()
	app.RequireStop()
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
