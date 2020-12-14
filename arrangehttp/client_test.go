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
	"github.com/xmidt-org/httpaux/roundtrip"
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
	suite.expected = nil
}

func (suite *ClientTestSuite) testHandleFunc(response http.ResponseWriter, request *http.Request) {
	for name, values := range suite.expected {
		suite.ElementsMatch(
			values,
			request.Header.Values(name),
			fmt.Sprintf("Header %s did not match", name),
		)
	}

	response.WriteHeader(299)
}

// newRequest creates a request to the test server
func (suite *ClientTestSuite) newRequest(h http.Header) *http.Request {
	request, err := http.NewRequest("GET", suite.server.URL+"/test", nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(request)

	for name, values := range h {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}

	return request
}

// checkClient makes a test request to our internal server
func (suite *ClientTestSuite) checkClient(client *http.Client, request *http.Request, expected http.Header) {
	suite.expected = expected
	response, err := client.Do(request)
	suite.Require().NoError(err)

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

	request := suite.newRequest(http.Header{"X-Test": {"true"}})
	suite.checkClient(
		client,
		request,
		http.Header{
			"X-Test": {"true"},
		},
	)

	app.RequireStop()
}

func (suite *ClientTestSuite) TestMiddleware() {
	var client *http.Client

	app := suite.Fxtest(
		fx.Provide(
			func() roundtrip.Constructor {
				return func(next http.RoundTripper) http.RoundTripper {
					return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
						request.Header.Set("X-Middleware-Unnamed", "true")
						return next.RoundTrip(request)
					})
				}
			},
			fx.Annotated{
				Group: "constructors",
				Target: func() roundtrip.Constructor {
					return func(next http.RoundTripper) http.RoundTripper {
						return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
							request.Header.Set("X-Middleware-Group1", "true")
							return next.RoundTrip(request)
						})
					}
				},
			},
			fx.Annotated{
				Group: "constructors",
				Target: func() roundtrip.Constructor {
					return func(next http.RoundTripper) http.RoundTripper {
						return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
							request.Header.Set("X-Middleware-Group2", "true")
							return next.RoundTrip(request)
						})
					}
				},
			},
			func() roundtrip.Chain {
				return roundtrip.NewChain(
					func(next http.RoundTripper) http.RoundTripper {
						return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
							request.Header.Set("X-Middleware-Unnamed-Chain", "true")
							return next.RoundTrip(request)
						})
					},
				)
			},
		),
		Client{
			Inject: arrange.Inject{
				struct {
					fx.In
					F1 roundtrip.Constructor
					F2 []roundtrip.Constructor `group:"constructors"`
					F3 roundtrip.Chain
				}{},
			},
			Middleware: roundtrip.NewChain(
				func(next http.RoundTripper) http.RoundTripper {
					return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
						request.Header.Set("X-Middleware-Option", "true")
						return next.RoundTrip(request)
					})
				},
			),
		}.Provide(),
		fx.Populate(&client),
	)

	suite.Require().NoError(app.Err())
	app.RequireStart()

	request := suite.newRequest(http.Header{"X-Test": {"true"}})
	suite.checkClient(
		client,
		request,
		http.Header{
			"X-Test":                     {"true"},
			"X-Middleware-Unnamed":       {"true"},
			"X-Middleware-Group1":        {"true"},
			"X-Middleware-Group2":        {"true"},
			"X-Middleware-Unnamed-Chain": {"true"},
			"X-Middleware-Option":        {"true"},
		},
	)
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
