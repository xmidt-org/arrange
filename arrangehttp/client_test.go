package arrangehttp

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetest"
	"github.com/xmidt-org/arrange/arrangetls"
	"go.uber.org/fx"
)

type simpleClientFactory struct {
	returnErr error
}

func (scf simpleClientFactory) NewClient() (*http.Client, error) {
	if scf.returnErr != nil {
		return nil, scf.returnErr
	}

	return new(http.Client), nil
}

type TransportConfigTestSuite struct {
	suite.Suite
}

func (suite *TransportConfigTestSuite) TestBasic() {
	tc := TransportConfig{
		TLSHandshakeTimeout:   15 * time.Second,
		DisableKeepAlives:     true,
		DisableCompression:    true,
		MaxIdleConns:          17,
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       92,
		IdleConnTimeout:       2 * time.Minute,
		ResponseHeaderTimeout: 13 * time.Millisecond,
		ExpectContinueTimeout: 29 * time.Second,
		ProxyConnectHeader: http.Header{
			"Something": []string{"Of Value"},
		},
		MaxResponseHeaderBytes: 347234,
		WriteBufferSize:        234867,
		ReadBufferSize:         93247,
		ForceAttemptHTTP2:      true,
	}

	transport, err := tc.NewTransport(nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(transport)

	suite.Nil(transport.TLSClientConfig)
	suite.Equal(15*time.Second, transport.TLSHandshakeTimeout)
	suite.True(transport.DisableKeepAlives)
	suite.True(transport.DisableCompression)
	suite.Equal(17, transport.MaxIdleConns)
	suite.Equal(5, transport.MaxIdleConnsPerHost)
	suite.Equal(92, transport.MaxConnsPerHost)
	suite.Equal(2*time.Minute, transport.IdleConnTimeout)
	suite.Equal(13*time.Millisecond, transport.ResponseHeaderTimeout)
	suite.Equal(29*time.Second, transport.ExpectContinueTimeout)
	suite.Equal(
		http.Header{"Something": []string{"Of Value"}},
		transport.ProxyConnectHeader,
	)
	suite.Equal(int64(347234), transport.MaxResponseHeaderBytes)
	suite.Equal(234867, transport.WriteBufferSize)
	suite.Equal(93247, transport.ReadBufferSize)
	suite.True(transport.ForceAttemptHTTP2)
}

func (suite *TransportConfigTestSuite) TestTLS() {
	var (
		tc TransportConfig

		config = arrangetls.Config{
			InsecureSkipVerify: true,
		}
	)

	transport, err := tc.NewTransport(&config)
	suite.Require().NoError(err)
	suite.Require().NotNil(transport)
	suite.NotNil(transport.TLSClientConfig)
}

func (suite *TransportConfigTestSuite) TestError() {
	var (
		tc TransportConfig

		config = arrangetls.Config{
			Certificates: arrangetls.ExternalCertificates{
				{
					CertificateFile: "missing",
					KeyFile:         "missing",
				},
			},
		}
	)

	transport, err := tc.NewTransport(&config)
	suite.Error(err)
	suite.NotNil(transport)
}

func TestTransportConfig(t *testing.T) {
	suite.Run(t, new(TransportConfigTestSuite))
}

type ClientTestSuite struct {
	arrangetest.Suite
	server   *httptest.Server
	expected http.Header
}

var _ suite.SetupAllSuite = (*ClientTestSuite)(nil)
var _ suite.SetupTestSuite = (*ClientTestSuite)(nil)
var _ suite.TearDownAllSuite = (*ClientTestSuite)(nil)

func (suite *ClientTestSuite) SetupSuite() {
	r := mux.NewRouter()
	r.HandleFunc("/test", suite.testHandleFunc)
	suite.server = httptest.NewServer(r)
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *ClientTestSuite) testHandleFunc(response http.ResponseWriter, request *http.Request) {
	for k, v := range suite.expected {
		suite.Equal(
			v,
			request.Header[k],
			fmt.Sprintf("Header %s did not match", k),
		)
	}

	response.WriteHeader(299)
}

// newRequest creates a request to the test server
func (suite *ClientTestSuite) newRequest(h http.Header) *http.Request {
	request, err := http.NewRequest("GET", suite.server.URL+"/test", nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(request)

	for k, values := range h {
		for _, v := range values {
			request.Header.Add(k, v)
		}
	}

	return request
}

// checkClient makes a test request to our internal server
func (suite *ClientTestSuite) checkClient(client *http.Client, expected http.Header) {
	request := suite.newRequest(expected)

	suite.expected = expected
	suite.T().Log("suite.expected", suite.expected)
	response, err := client.Do(request)
	suite.Require().NoError(err)
	suite.expected = nil

	suite.Require().NotNil(response)
	defer suite.NoError(response.Body.Close())
	_, err = io.Copy(ioutil.Discard, response.Body)
	suite.Require().NoError(err)

	suite.Equal(299, response.StatusCode, "the server did not process the request")
}

func (suite *ClientTestSuite) TestClientFactoryError() {
	var client *http.Client

	app := suite.Fx(
		Client{
			ClientFactory: simpleClientFactory{
				returnErr: errors.New("expected ClientFactory error"),
			},
		}.Provide(),
		fx.Populate(&client),
	)

	suite.Error(app.Err())
}

func (suite *ClientTestSuite) TestConfigureError() {
	var client *http.Client

	app := suite.Fx(
		Client{
			Options: arrange.Invoke{
				func(c *http.Client) error {
					suite.NotNil(c)
					return errors.New("expected")
				},
			},
		}.Provide(),
		fx.Populate(&client),
	)

	suite.Error(app.Err())
}

func (suite *ClientTestSuite) TestDefaults() {
	var client *http.Client

	app := suite.Fxtest(
		Client{}.Provide(),
		fx.Populate(&client),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()

	suite.Require().NotNil(client)
	suite.checkClient(client, http.Header{
		"X-Test": {"true"},
	})

	app.RequireStop()
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
