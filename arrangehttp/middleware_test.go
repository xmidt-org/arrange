package arrangehttp

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MiddlewareSuite struct {
	suite.Suite
}

func (suite *MiddlewareSuite) testApplyMiddleware(count int) {
	current := count - 1 // middleware themselves run in reverse order
	middleware := make([]func(http.Handler) http.Handler, 0, count)
	for i := 0; i < count; i++ {
		i := i
		middleware = append(middleware, func(actual http.Handler) http.Handler {
			suite.Same(http.DefaultServeMux, actual)
			suite.Equal(i, current)
			current--
			return actual
		})
	}

	suite.Equal(
		http.DefaultServeMux,
		ApplyMiddleware[http.Handler](http.DefaultServeMux, middleware...),
	)
}

func (suite *MiddlewareSuite) TestApplyMiddleware() {
	for _, count := range []int{0, 1, 2, 5} {
		suite.Run(fmt.Sprintf("count=%d", count), func() {
			suite.testApplyMiddleware(count)
		})
	}
}

func TestMiddleware(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}
