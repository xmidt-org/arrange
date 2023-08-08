package arrangeoption

import (
	"errors"
	"fmt"
	"github.com/xmidt-org/arrange/arrangetest"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AsOptionSuite[T any] struct {
	arrangetest.OptionSuite[T]
}

func (suite *AsOptionSuite[T]) TestClosure() {
	expected := new(mockOption[T])
	wrapper := AsOption[T](expected.Apply)
	suite.Require().NotNil(wrapper)

	expected.ExpectApply(suite.Target).Return(nil)
	suite.NoError(wrapper.Apply(suite.Target))
	expected.AssertExpectations(suite.T())
}

func (suite *AsOptionSuite[T]) TestClosureNoError() {
	expected := new(mockOptionNoError[T])
	wrapper := AsOption[T](expected.Apply)
	suite.Require().NotNil(wrapper)

	expected.ExpectApply(suite.Target)
	suite.NoError(wrapper.Apply(suite.Target))
	expected.AssertExpectations(suite.T())
}

func TestAsOptionServer(t *testing.T) {
	suite.Run(t, new(AsOptionSuite[http.Server]))
}

func TestAsOptionClient(t *testing.T) {
	suite.Run(t, new(AsOptionSuite[http.Client]))
}

type ApplyOptionsSuite[T any] struct {
	arrangetest.OptionSuite[T]
}

func (suite *ApplyOptionsSuite[T]) testApplyOptions(count int) {
	var (
		current = 0
		opts    = make(Options[T], 0, count)
	)

	for i := 0; i < count; i++ {
		i := i
		opts = append(opts, AsOption[T](func(actual *T) {
			suite.NotNil(actual)
			suite.Same(suite.Target, actual)
			suite.Equal(i, current)
			current++
		}))
	}

	actual, err := ApplyOptions(suite.Target, opts...)
	suite.Same(suite.Target, actual)
	suite.NoError(err)
}

func (suite *ApplyOptionsSuite[T]) TestApplyOptions() {
	for _, count := range []int{0, 1, 2, 5} {
		suite.Run(fmt.Sprintf("count=%d", count), func() {
			suite.testApplyOptions(count)
		})
	}
}

func TestApplyOptionsServer(t *testing.T) {
	suite.Run(t, new(ApplyOptionsSuite[http.Server]))
}

func TestApplyOptionsClient(t *testing.T) {
	suite.Run(t, new(ApplyOptionsSuite[http.Server]))
}

func TestInvalidOption(t *testing.T) {
	expected := errors.New("expected")
	assert.Equal(
		t,
		expected,
		InvalidOption[http.Server](expected).Apply(nil),
	)
}
