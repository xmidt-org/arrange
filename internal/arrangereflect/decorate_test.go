package arrangereflect

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DecorateSuite struct {
	suite.Suite
}

func (suite *DecorateSuite) TestNoDecorators() {
	suite.Equal(
		http.DefaultServeMux,
		Decorate[http.Handler, func(http.Handler) http.Handler](http.DefaultServeMux),
	)
}

func (suite *DecorateSuite) testDecorators(count int) {
	current := count - 1 // decorators themselves run in reverse order
	decorators := make([]func(http.Handler) http.Handler, 0, count)
	for i := 0; i < count; i++ {
		i := i
		decorators = append(decorators, func(actual http.Handler) http.Handler {
			suite.Same(http.DefaultServeMux, actual)
			suite.Equal(i, current)
			current--
			return actual
		})
	}

	suite.Equal(
		http.DefaultServeMux,
		Decorate[http.Handler, func(http.Handler) http.Handler](http.DefaultServeMux, decorators...),
	)
}

func (suite *DecorateSuite) TestDecorators() {
	for _, decoratorCount := range []int{1, 2, 5} {
		suite.Run(fmt.Sprintf("decoratorCount=%d", decoratorCount), func() {
			suite.testDecorators(decoratorCount)
		})
	}
}

func TestDecorate(t *testing.T) {
	suite.Run(t, new(DecorateSuite))
}
