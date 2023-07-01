package arrangehttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetls"
)

type TransportConfigSuite struct {
	arrangetls.Suite
}

func (suite *TransportConfigSuite) assertTransport(expected TransportConfig, actual *http.Transport) {
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

func (suite *TransportConfigSuite) TestTransportConfig() {
	expected := TransportConfig{
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

	suite.Run("NoTLS", func() {
		actual, err := expected.NewTransport(nil)
		suite.Require().NoError(err)
		suite.Require().NotNil(actual)
		suite.assertTransport(expected, actual)
		suite.Nil(actual.TLSClientConfig)
	})

	suite.Run("TLS", func() {
		actual, err := expected.NewTransport(suite.Config())
		suite.Require().NoError(err)
		suite.Require().NotNil(actual)
		suite.assertTransport(expected, actual)
		suite.Require().NotNil(actual.TLSClientConfig)
	})
}

func TestTransportConfig(t *testing.T) {
	suite.Run(t, new(TransportConfigSuite))
}
