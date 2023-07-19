package arrangehttp

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetest"
	"github.com/xmidt-org/arrange/arrangetls"
)

type ListenerSuite struct {
	arrangetls.Suite
}

func (suite *ListenerSuite) testDefaultListenerFactoryBasic() {
	var (
		factory DefaultListenerFactory
		server  = &http.Server{
			Addr: ":0",
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	suite.Require().NoError(err)
	suite.Require().NotNil(listener)
	suite.NotNil(listener.Addr())
	listener.Close()
}

func (suite *ListenerSuite) testDefaultListenerFactoryWithTLS() {
	var (
		factory DefaultListenerFactory
		server  = &http.Server{
			Addr:      ":0",
			TLSConfig: suite.TLSConfig(),
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	suite.Require().NoError(err)
	suite.Require().NotNil(listener)
	suite.NotNil(listener.Addr())
	listener.Close()
}

func (suite *ListenerSuite) testDefaultListenerFactoryError() {
	var (
		factory = DefaultListenerFactory{
			Network: "this is a bad network",
		}

		server = &http.Server{
			Addr: ":0",
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	suite.Error(err)

	if !suite.Nil(listener) {
		// cleanup if the assertion fails, meaning the factory incorrectly
		// returned a non-nil listener AND a non-nil error.
		listener.Close()
	}
}

func (suite *ListenerSuite) TestDefaultListenerFactory() {
	suite.Run("Basic", suite.testDefaultListenerFactoryBasic)
	suite.Run("WithTLS", suite.testDefaultListenerFactoryWithTLS)
	suite.Run("Error", suite.testDefaultListenerFactoryError)
}

func (suite *ListenerSuite) testNewListenerNilListenerFactory() {
	var (
		capture = make(chan net.Addr, 1)
		l, err  = NewListener(
			context.Background(),
			nil,
			&http.Server{
				Addr: ":0",
			},
			arrangetest.ListenCapture(capture),
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()
	actual := arrangetest.ListenReceive(suite, capture, time.Second)
	suite.Equal(l.Addr(), actual)
}

func (suite *ListenerSuite) testNewListenerCustomListenerFactory() {
	var (
		capture = make(chan net.Addr, 1)
		l, err  = NewListener(
			context.Background(),
			ServerConfig{},
			&http.Server{
				Addr: ":0",
			},
			arrangetest.ListenCapture(capture),
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()
	actual := arrangetest.ListenReceive(suite, capture, time.Second)
	suite.Equal(l.Addr(), actual)
}

func (suite *ListenerSuite) TestNewListener() {
	suite.Run("NilListenerFactory", suite.testNewListenerNilListenerFactory)
	suite.Run("CustomListenerFactory", suite.testNewListenerCustomListenerFactory)
}

func TestListener(t *testing.T) {
	suite.Run(t, new(ListenerSuite))
}
