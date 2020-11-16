package arrange

import (
	"errors"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"
	"go.uber.org/multierr"
)

type InvokeTestSuite struct {
	suite.Suite
}

func (suite *InvokeTestSuite) TestEmpty() {
	suite.NoError(
		Invoke{}.Apply(),
	)
}

func (suite *InvokeTestSuite) TestTypicalUsage() {
	type Custom *mux.Router

	var called []int
	router := mux.NewRouter()
	invoke := Invoke{
		func(actual *mux.Router) {
			called = append(called, 1)
			suite.True(router == actual)
		},
		func(actual *mux.Router) error {
			called = append(called, 2)
			suite.True(router == actual)
			return nil
		},
		func(actual Custom) { // force a conversion to happen
			called = append(called, 3)
			suite.True(router == actual)
		},
	}

	suite.NoError(invoke.Apply(router))
	suite.Equal([]int{1, 2, 3}, called)
}

func (suite *InvokeTestSuite) TestErrors() {
	var called []int
	expectedErr := errors.New("expected")
	router := mux.NewRouter()
	invoke := Invoke{
		func(actual *mux.Router) {
			called = append(called, 1)
			suite.True(router == actual)
		},
		func(actual *mux.Router) error {
			called = append(called, 2)
			suite.True(router == actual)
			return expectedErr
		},
	}

	err := invoke.Apply(router)
	suite.Equal([]int{1, 2}, called)

	suite.Require().Error(err)
	suite.Equal(expectedErr, err)
}

func (suite *InvokeTestSuite) TestNotAFunction() {
	var called []int
	router := mux.NewRouter()
	invoke := Invoke{
		"this is not a function",
		func(actual *mux.Router) error {
			called = append(called, 1)
			suite.True(router == actual)
			return nil
		},
	}

	err := invoke.Apply(router)
	suite.Equal([]int{1}, called)

	suite.Require().Error(err)
	suite.IsType(new(InvokeError), err)
	suite.Contains(err.Error(), "INVOKE ERROR")
}

func (suite *InvokeTestSuite) TestWrongNumberOfInputs() {
	var called []int
	router := mux.NewRouter()
	invoke := Invoke{
		func() {
			// bad
			suite.Fail("This closure should not have been called")
		},
		func(actual *mux.Router) error {
			called = append(called, 1)
			suite.True(router == actual)
			return nil
		},
		func(a, b *mux.Router) error {
			// also bad
			suite.Fail("This closure should not have been called")
			return nil
		},
	}

	err := invoke.Apply(router)
	suite.Equal([]int{1}, called)

	suite.Require().Error(err)
	suite.Contains(err.Error(), "INVOKE ERROR")
	errs := multierr.Errors(err)
	suite.Require().Len(errs, 2)
	suite.IsType(new(InvokeError), errs[0])
	suite.IsType(new(InvokeError), errs[1])
}

func (suite *InvokeTestSuite) TestTooManyReturnValues() {
	var called []int
	router := mux.NewRouter()
	invoke := Invoke{
		func(*mux.Router) (int, error) {
			// bad
			suite.Fail("This closure should not have been called")
			return 0, nil
		},
		func(actual *mux.Router) error {
			called = append(called, 1)
			suite.True(router == actual)
			return nil
		},
	}

	err := invoke.Apply(router)
	suite.Equal([]int{1}, called)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "INVOKE ERROR")
	suite.IsType(new(InvokeError), err)
}

func (suite *InvokeTestSuite) TestWrongInputType() {
	var called []int
	router := mux.NewRouter()
	invoke := Invoke{
		func(string) {
			// bad
			suite.Fail("This closure should not have been called")
		},
		func(actual *mux.Router) error {
			called = append(called, 1)
			suite.True(router == actual)
			return nil
		},
	}

	err := invoke.Apply(router)
	suite.Equal([]int{1}, called)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "INVOKE ERROR")
	suite.IsType(new(InvokeError), err)
}

func (suite *InvokeTestSuite) TestNonErrorReturnValue() {
	var called []int
	router := mux.NewRouter()
	invoke := Invoke{
		func(*mux.Router) int {
			// bad
			suite.Fail("This closure should not have been called")
			return 0
		},
		func(actual *mux.Router) error {
			called = append(called, 1)
			suite.True(router == actual)
			return nil
		},
	}

	err := invoke.Apply(router)
	suite.Equal([]int{1}, called)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "INVOKE ERROR")
	suite.IsType(new(InvokeError), err)
}

func TestInvoke(t *testing.T) {
	suite.Run(t, new(InvokeTestSuite))
}
