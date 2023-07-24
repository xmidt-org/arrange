package arrangehttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/roundtrip"
)

type ClientOptionSuite struct {
	OptionSuite[http.Client]
}

func (suite *ClientOptionSuite) testClientMiddlewareNoTransport() {
	called := false

	ClientMiddleware(func(next http.RoundTripper) http.RoundTripper {
		suite.Same(http.DefaultTransport, next)
		return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
			called = true
			return nil, nil
		})
	}).Apply(suite.target)

	suite.Require().NotNil(suite.target.Transport)
	suite.target.Transport.RoundTrip(new(http.Request))
	suite.True(called)
}

func (suite *ClientOptionSuite) testClientMiddlewareWithTransport() {
	expectedRequest := httptest.NewRequest("GET", "/", nil)

	suite.target.Transport = roundtrip.Func(func(actualRequest *http.Request) (*http.Response, error) {
		suite.Same(expectedRequest, actualRequest)
		return &http.Response{
			Header: http.Header{
				"Custom": []string{"true"},
			},
		}, nil
	})

	ClientMiddleware(func(next http.RoundTripper) http.RoundTripper {
		suite.Require().NotNil(next)
		return roundtrip.Func(func(request *http.Request) (*http.Response, error) {
			response, err := next.RoundTrip(request)
			suite.Require().NoError(err)
			suite.Require().NotNil(response)

			response.Header.Set("Middleware", "true")
			return response, err
		})
	}).Apply(suite.target)

	suite.Require().NotNil(suite.target.Transport)

	response, err := suite.target.Transport.RoundTrip(expectedRequest)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Equal(
		"true",
		response.Header.Get("Custom"),
	)

	suite.Equal(
		"true",
		response.Header.Get("Middleware"),
	)
}

func (suite *ClientOptionSuite) TestClientMiddleware() {
	suite.Run("NoTransport", suite.testClientMiddlewareNoTransport)
	suite.Run("WithTransport", suite.testClientMiddlewareWithTransport)
}

func TestClientOption(t *testing.T) {
	suite.Run(t, new(ClientOptionSuite))
}
