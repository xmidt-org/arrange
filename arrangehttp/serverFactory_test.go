package arrangehttp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetls"
)

type ServerConfigSuite struct {
	arrangetls.Suite
}

// expectedServerConfig returns a non-TLS ServerConfig with relevant server fields set to
// distinct, non-default values.
func (suite *ServerConfigSuite) expectedServerConfig() ServerConfig {
	return ServerConfig{
		Address:           ":1234",
		ReadTimeout:       123 * time.Minute,
		ReadHeaderTimeout: 49 * time.Second,
		WriteTimeout:      284 * time.Millisecond,
		IdleTimeout:       28 * time.Minute,
		MaxHeaderBytes:    319831,
	}
}

// assertServerConfig verifies that an *http.Server was created properly from the given ServerConfig.
// This method does nothing with the *tls.Config.
func (suite *ServerConfigSuite) assertServerConfig(expected ServerConfig, actual *http.Server) {
	suite.Equal(expected.Address, actual.Addr)
	suite.Equal(expected.ReadTimeout, actual.ReadTimeout)
	suite.Equal(expected.ReadHeaderTimeout, actual.ReadHeaderTimeout)
	suite.Equal(expected.WriteTimeout, actual.WriteTimeout)
	suite.Equal(expected.IdleTimeout, actual.IdleTimeout)
	suite.Equal(expected.MaxHeaderBytes, actual.MaxHeaderBytes)
}

func (suite *ServerConfigSuite) testNewServerNoTLS() {
	sc := suite.expectedServerConfig()
	server, err := sc.NewServer()
	suite.Require().NoError(err)
	suite.Require().NotNil(server)
	suite.assertServerConfig(sc, server)
	suite.Nil(server.TLSConfig)
}

func (suite *ServerConfigSuite) testNewServerTLS() {
	sc := suite.expectedServerConfig()
	sc.TLS = suite.Config()

	server, err := sc.NewServer()
	suite.Require().NoError(err)
	suite.Require().NotNil(server)
	suite.NotNil(server.TLSConfig)
}

func (suite *ServerConfigSuite) TestNewServer() {
	suite.Run("NoTLS", suite.testNewServerNoTLS)
	suite.Run("TLS", suite.testNewServerTLS)
}

func (suite *ServerConfigSuite) testListenDefault() {
	var (
		s = &http.Server{
			Addr: ":0",
		}

		// all defaults in the ServerConfig
		l, err = ServerConfig{}.Listen(
			context.Background(), s,
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()

	suite.IsType((*net.TCPListener)(nil), l)
}

func (suite *ServerConfigSuite) testListenNoTLS() {
	var (
		s = &http.Server{
			Addr: ":0",
		}

		l, err = ServerConfig{
			Network:   "tcp",
			KeepAlive: 2 * time.Minute,
		}.Listen(
			context.Background(), s,
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()

	suite.IsType((*net.TCPListener)(nil), l)
}

func (suite *ServerConfigSuite) testListenTLS() {
	var (
		s = &http.Server{
			Addr:      ":0",
			TLSConfig: suite.TLSConfig(),
		}

		l, err = ServerConfig{
			KeepAlive: time.Minute,
		}.Listen(
			context.Background(), s,
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()

	_, isTCP := l.(*net.TCPListener)
	suite.False(isTCP)
}

func (suite *ServerConfigSuite) TestListen() {
	suite.Run("Default", suite.testListenDefault)
	suite.Run("NoTLS", suite.testListenNoTLS)
	suite.Run("TLS", suite.testListenTLS)
}

func (suite *ServerConfigSuite) testApplyNoHeader() {
	var (
		s = &http.Server{
			Handler: http.DefaultServeMux,
		}
	)

	suite.NoError(ServerConfig{}.Apply(s))
	suite.Require().NotNil(s.Handler)
	suite.Same(http.DefaultServeMux, s.Handler) // no decoration should have occurred
}

func (suite *ServerConfigSuite) testApplyNoHandler() {
	var (
		s  = new(http.Server) // no handler set
		sc = ServerConfig{
			Header: http.Header{
				"Custom": []string{"true"},
			},
		}
	)

	suite.NoError(sc.Apply(s))
	suite.Require().NotNil(s.Handler)

	var (
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	s.Handler.ServeHTTP(response, request)
	suite.Equal(http.StatusNotFound, response.Result().StatusCode) // http.DefaultServeMux
	suite.Equal("true", response.Result().Header.Get("Custom"))
}

func (suite *ServerConfigSuite) testApplyWithHandler() {
	var (
		s = &http.Server{
			Handler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(299)
			}),
		}

		sc = ServerConfig{
			Header: http.Header{
				"Custom": []string{"true"},
			},
		}
	)

	suite.NoError(sc.Apply(s))
	suite.Require().NotNil(s.Handler)

	var (
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	s.Handler.ServeHTTP(response, request)
	suite.Equal(299, response.Result().StatusCode)
	suite.Equal("true", response.Result().Header.Get("Custom"))
}

func (suite *ServerConfigSuite) TestApply() {
	suite.Run("NoHeader", suite.testApplyNoHeader)
	suite.Run("NoHandler", suite.testApplyNoHandler)
	suite.Run("WithHandler", suite.testApplyWithHandler)
}

func TestServerConfig(t *testing.T) {
	suite.Run(t, new(ServerConfigSuite))
}
