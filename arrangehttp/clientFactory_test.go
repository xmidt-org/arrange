package arrangehttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetls"
)

type ClientConfigSuite struct {
	arrangetls.Suite
}

// expectedTransportConfig returns a TransportConfig with everything set to a distinct,
// non-default value.
func (suite *ClientConfigSuite) expectedTransportConfig() TransportConfig {
	return TransportConfig{
		TLSHandshakeTimeout:   20 * time.Second,
		DisableKeepAlives:     true,
		DisableCompression:    true,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       12,
		IdleConnTimeout:       13 * time.Minute,
		ResponseHeaderTimeout: 12 * time.Hour,
		ExpectContinueTimeout: 5 * time.Second,
		ProxyConnectHeader: http.Header{
			"Test": []string{"Value"},
		},
		MaxResponseHeaderBytes: 4096,
		WriteBufferSize:        1123,
		ReadBufferSize:         9473,
		ForceAttemptHTTP2:      true,
	}
}

// expectedClientConfig creates a ClientConfig with everything set to distinct, non-default
// values.  The given TLS config is optional and may be nil.
func (suite *ClientConfigSuite) expectedClientConfig(tls *arrangetls.Config) ClientConfig {
	return ClientConfig{
		Timeout:   457 * time.Millisecond,
		Transport: suite.expectedTransportConfig(),
		Header: http.Header{
			"Custom": []string{"true"},
		},
		TLS: tls,
	}
}

// assertTransport asserts that an *http.Transport was correctly created from a TransportConfig.
func (suite *ClientConfigSuite) assertTransport(expected TransportConfig, actual *http.Transport) {
	suite.Equal(expected.TLSHandshakeTimeout, actual.TLSHandshakeTimeout)
	suite.Equal(expected.DisableKeepAlives, actual.DisableKeepAlives)
	suite.Equal(expected.DisableCompression, actual.DisableCompression)
	suite.Equal(expected.MaxIdleConns, actual.MaxIdleConns)
	suite.Equal(expected.MaxIdleConnsPerHost, actual.MaxIdleConnsPerHost)
	suite.Equal(expected.MaxConnsPerHost, actual.MaxConnsPerHost)
	suite.Equal(expected.IdleConnTimeout, actual.IdleConnTimeout)
	suite.Equal(expected.ResponseHeaderTimeout, actual.ResponseHeaderTimeout)
	suite.Equal(expected.ExpectContinueTimeout, actual.ExpectContinueTimeout)
	suite.Equal(expected.ProxyConnectHeader, actual.ProxyConnectHeader)
	suite.Equal(expected.MaxResponseHeaderBytes, actual.MaxResponseHeaderBytes)
	suite.Equal(expected.WriteBufferSize, actual.WriteBufferSize)
	suite.Equal(expected.ReadBufferSize, actual.ReadBufferSize)
	suite.Equal(expected.ForceAttemptHTTP2, actual.ForceAttemptHTTP2)
}

func (suite *ClientConfigSuite) assertClient(expected ClientConfig, actual *http.Client) {
	suite.Equal(expected.Timeout, actual.Timeout)
}

func (suite *ClientConfigSuite) testTransportConfigNoTLS() {
	var (
		expected    = suite.expectedTransportConfig()
		actual, err = expected.NewTransport(nil)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(actual)
	suite.assertTransport(expected, actual)
	suite.Nil(actual.TLSClientConfig)
}

func (suite *ClientConfigSuite) testTransportConfigTLS() {
	var (
		expected    = suite.expectedTransportConfig()
		actual, err = expected.NewTransport(suite.Config())
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(actual)
	suite.assertTransport(expected, actual)
	suite.Require().NotNil(actual.TLSClientConfig)
}

func (suite *ClientConfigSuite) TestTransportConfig() {
	suite.Run("NoTLS", suite.testTransportConfigNoTLS)
	suite.Run("TLS", suite.testTransportConfigTLS)
}

func (suite *ClientConfigSuite) TestNewClient() {
}

func TestClientConfig(t *testing.T) {
	suite.Run(t, new(ClientConfigSuite))
}
