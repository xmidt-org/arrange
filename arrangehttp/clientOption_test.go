package arrangehttp

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/multierr"
)

type AsClientOptionSuite struct {
	suite.Suite
	expectedClient *http.Client
}

func (suite *AsClientOptionSuite) SetupTest() {
	suite.expectedClient = new(http.Client)
}

func (suite *AsClientOptionSuite) TestInvalidType() {
	o := AsClientOption(123)
	suite.Require().NotNil(o)

	var expectedErr *InvalidClientOptionTypeError
	suite.ErrorAs(o.ApplyToClient(suite.expectedClient), &expectedErr)
	suite.Require().NotNil(expectedErr)
	suite.Require().NotNil(expectedErr.Type)
	suite.NotEmpty(expectedErr.Error())
}

func (suite *AsClientOptionSuite) TestTrivial() {
	expected := new(mockOption)
	suite.Same(expected, AsClientOption(expected))
	expected.AssertExpectations(suite.T())
}

func (suite *AsClientOptionSuite) TestApplyToClientNoError() {
	expected := new(mockOptionNoError)
	wrapper := AsClientOption(expected)
	suite.Require().NotNil(wrapper)

	expected.ExpectApplyToClient(suite.expectedClient)
	suite.NoError(wrapper.ApplyToClient(suite.expectedClient))
	expected.AssertExpectations(suite.T())
}

func (suite *AsClientOptionSuite) TestClosure() {
	expected := new(mockOption)
	wrapper := AsClientOption(expected.ApplyToClient)
	suite.Require().NotNil(wrapper)

	expected.ExpectApplyToClient(suite.expectedClient).Return(nil)
	suite.NoError(wrapper.ApplyToClient(suite.expectedClient))
	expected.AssertExpectations(suite.T())
}

func (suite *AsClientOptionSuite) TestClosureNoError() {
	expected := new(mockOptionNoError)
	wrapper := AsClientOption(expected.ApplyToClient)
	suite.Require().NotNil(wrapper)

	expected.ExpectApplyToClient(suite.expectedClient)
	suite.NoError(wrapper.ApplyToClient(suite.expectedClient))
	expected.AssertExpectations(suite.T())
}

func TestAsClientOption(t *testing.T) {
	suite.Run(t, new(AsClientOptionSuite))
}

type ClientOptionSuite struct {
	suite.Suite
	expectedClient *http.Client
}

func (suite *ClientOptionSuite) SetupTest() {
	suite.expectedClient = new(http.Client)
}

func (suite *ClientOptionSuite) SetupSubTest() {
	suite.expectedClient = new(http.Client)
}

func (suite *ClientOptionSuite) testClientOptionsEmpty() {
	suite.NoError(
		ClientOptions{}.ApplyToClient(suite.expectedClient),
	)
}

func (suite *ClientOptionSuite) testClientOptionsAllSuccess() {
	mocks := []*mockOption{
		new(mockOption),
		new(mockOption),
		new(mockOption),
	}

	mocks[0].ExpectApplyToClient(suite.expectedClient).Return(nil)
	mocks[1].ExpectApplyToClient(suite.expectedClient).Return(nil)
	mocks[2].ExpectApplyToClient(suite.expectedClient).Return(nil)

	suite.NoError(
		ClientOptions{
			mocks[0], mocks[1], mocks[2],
		}.ApplyToClient(suite.expectedClient),
	)

	mocks[0].AssertExpectations(suite.T())
	mocks[1].AssertExpectations(suite.T())
	mocks[2].AssertExpectations(suite.T())
}

func (suite *ClientOptionSuite) testClientOptionsAllFail() {
	mocks := []*mockOption{
		new(mockOption),
		new(mockOption),
		new(mockOption),
	}

	expectedErrors := []error{
		errors.New("expected 0"),
		errors.New("expected 1"),
		errors.New("expected 2"),
	}

	mocks[0].ExpectApplyToClient(suite.expectedClient).Return(expectedErrors[0])
	mocks[1].ExpectApplyToClient(suite.expectedClient).Return(expectedErrors[1])
	mocks[2].ExpectApplyToClient(suite.expectedClient).Return(expectedErrors[2])

	actualErrors := multierr.Errors(
		ClientOptions{
			mocks[0], mocks[1], mocks[2],
		}.ApplyToClient(suite.expectedClient),
	)

	suite.Require().Len(actualErrors, len(expectedErrors))
	suite.Same(expectedErrors[0], actualErrors[0])
	suite.Same(expectedErrors[1], actualErrors[1])
	suite.Same(expectedErrors[2], actualErrors[2])

	mocks[0].AssertExpectations(suite.T())
	mocks[1].AssertExpectations(suite.T())
	mocks[2].AssertExpectations(suite.T())
}

func (suite *ClientOptionSuite) TestClientOptions() {
	suite.Run("Empty", suite.testClientOptionsEmpty)
	suite.Run("AllSuccess", suite.testClientOptionsAllSuccess)
	suite.Run("AllFail", suite.testClientOptionsAllFail)
}

func TestClientOption(t *testing.T) {
	suite.Run(t, new(ClientOptionSuite))
}
