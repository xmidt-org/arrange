package arrangehttp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetls"
)

type DefaultListenerFactorySuite struct {
	arrangetls.Suite
}

func (suite *DefaultListenerFactorySuite) TestBasic() {
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

func (suite *DefaultListenerFactorySuite) TestTLS() {
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

func (suite *DefaultListenerFactorySuite) TestError() {
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

func TestDefaultListenerFactory(t *testing.T) {
	suite.Run(t, new(DefaultListenerFactorySuite))
}
