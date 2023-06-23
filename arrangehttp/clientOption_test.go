package arrangehttp

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ClientOptionSuite struct {
	OptionSuite[http.Client]
}

func (suite *ClientOptionSuite) testClientMiddleware(initialTransport http.RoundTripper, count int) *http.Response {
	var (
		current    = 0
		middleware []func(http.RoundTripper) http.RoundTripper
		c          = &http.Client{
			Transport: initialTransport,
		}
	)

	for i := 0; i < count; i++ {
		i := i
		middleware = append(middleware, func(next http.RoundTripper) http.RoundTripper {
			suite.Require().NotNil(next)
			return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
				suite.Equal(current, i)
				current++
				response, err := next.RoundTrip(request)
				suite.Require().NoError(err)
				response.Header.Set(fmt.Sprintf("Middleware-%d", i), "true")
				return response, err
			})
		})
	}

	ClientMiddleware(middleware...).Apply(c)
	suite.Require().NotNil(c.Transport)

	response, err := c.Transport.RoundTrip(new(http.Request))
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Equal(count, current)
	for i := 0; i < count; i++ {
		suite.Equal(
			"true",
			response.Header.Get(fmt.Sprintf("Middleware-%d", i)),
		)
	}

	return response
}

func (suite *ClientOptionSuite) TestClientMiddleware() {
}

func TestClientOption(t *testing.T) {
	suite.Run(t, new(ClientOptionSuite))
}
