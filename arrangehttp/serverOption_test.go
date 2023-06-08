package arrangehttp

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/multierr"
)

type AsServerOptionSuite struct {
	suite.Suite
	expectedServer *http.Server
}

func (suite *AsServerOptionSuite) SetupTest() {
	suite.expectedServer = new(http.Server)
}

func (suite *AsServerOptionSuite) TestInvalidType() {
	o := AsServerOption(123)
	suite.Require().NotNil(o)

	var expectedErr *InvalidServerOptionTypeError
	suite.ErrorAs(o.Apply(suite.expectedServer), &expectedErr)
	suite.Require().NotNil(expectedErr)
	suite.Require().NotNil(expectedErr.Type)
	suite.NotEmpty(expectedErr.Error())
}

func (suite *AsServerOptionSuite) TestTrivial() {
	expected := new(mockServerOption)
	suite.Same(expected, AsServerOption(expected))
	expected.AssertExpectations(suite.T())
}

func (suite *AsServerOptionSuite) TestApplyNoError() {
	expected := new(mockServerOptionNoError)
	wrapper := AsServerOption(expected)
	suite.Require().NotNil(wrapper)

	expected.ExpectApply(suite.expectedServer)
	suite.NoError(wrapper.Apply(suite.expectedServer))
	expected.AssertExpectations(suite.T())
}

func (suite *AsServerOptionSuite) TestClosure() {
	expected := new(mockServerOption)
	wrapper := AsServerOption(expected.Apply)
	suite.Require().NotNil(wrapper)

	expected.ExpectApply(suite.expectedServer).Return(nil)
	suite.NoError(wrapper.Apply(suite.expectedServer))
	expected.AssertExpectations(suite.T())
}

func (suite *AsServerOptionSuite) TestClosureNoError() {
	expected := new(mockServerOptionNoError)
	wrapper := AsServerOption(expected.Apply)
	suite.Require().NotNil(wrapper)

	expected.ExpectApply(suite.expectedServer)
	suite.NoError(wrapper.Apply(suite.expectedServer))
	expected.AssertExpectations(suite.T())
}

func TestAsServerOption(t *testing.T) {
	suite.Run(t, new(AsServerOptionSuite))
}

type ServerOptionSuite struct {
	suite.Suite
	expectedServer *http.Server
}

func (suite *ServerOptionSuite) SetupTest() {
	suite.expectedServer = new(http.Server)
}

func (suite *ServerOptionSuite) SetupSubTest() {
	suite.expectedServer = new(http.Server)
}

func (suite *ServerOptionSuite) TestServerOptionFunc() {
	suite.Run("NoError", func() {
		f := ServerOptionFunc(func(actual *http.Server) error {
			suite.Same(suite.expectedServer, actual)
			return nil
		})

		suite.NoError(f(suite.expectedServer))
	})

	suite.Run("Error", func() {
		expectedErr := errors.New("expected")
		f := ServerOptionFunc(func(actual *http.Server) error {
			suite.Same(suite.expectedServer, actual)
			return expectedErr
		})

		suite.Same(
			expectedErr,
			f(suite.expectedServer),
		)
	})
}

func (suite *ServerOptionSuite) TestServerOptions() {
	suite.Run("Empty", func() {
		suite.NoError(
			ServerOptions{}.Apply(suite.expectedServer),
		)
	})

	suite.Run("AllSuccess", func() {
		mocks := []*mockServerOption{
			new(mockServerOption),
			new(mockServerOption),
			new(mockServerOption),
		}

		mocks[0].ExpectApply(suite.expectedServer).Return(nil)
		mocks[1].ExpectApply(suite.expectedServer).Return(nil)
		mocks[2].ExpectApply(suite.expectedServer).Return(nil)

		suite.NoError(
			ServerOptions{
				mocks[0], mocks[1], mocks[2],
			}.Apply(suite.expectedServer),
		)

		mocks[0].AssertExpectations(suite.T())
		mocks[1].AssertExpectations(suite.T())
		mocks[2].AssertExpectations(suite.T())
	})

	suite.Run("AllFail", func() {
		mocks := []*mockServerOption{
			new(mockServerOption),
			new(mockServerOption),
			new(mockServerOption),
		}

		expectedErrors := []error{
			errors.New("expected 0"),
			errors.New("expected 1"),
			errors.New("expected 2"),
		}

		mocks[0].ExpectApply(suite.expectedServer).Return(expectedErrors[0])
		mocks[1].ExpectApply(suite.expectedServer).Return(expectedErrors[1])
		mocks[2].ExpectApply(suite.expectedServer).Return(expectedErrors[2])

		actualErrors := multierr.Errors(
			ServerOptions{
				mocks[0], mocks[1], mocks[2],
			}.Apply(suite.expectedServer),
		)

		suite.Require().Len(actualErrors, len(expectedErrors))
		suite.Same(expectedErrors[0], actualErrors[0])
		suite.Same(expectedErrors[1], actualErrors[1])
		suite.Same(expectedErrors[2], actualErrors[2])

		mocks[0].AssertExpectations(suite.T())
		mocks[1].AssertExpectations(suite.T())
		mocks[2].AssertExpectations(suite.T())
	})

	suite.Run("Add", func() {
		var (
			mock0 = new(mockServerOption)
			mock1 = new(mockServerOption)
			mock2 = new(mockServerOptionNoError)
			mock3 = new(mockServerOptionNoError)

			so ServerOptions
		)

		so.Add()
		suite.Empty(so)

		so.Add(mock0, mock1.Apply, mock2, mock3.Apply)
		suite.Require().Len(so, 4)

		mock0.ExpectApply(suite.expectedServer).Return(nil)
		mock1.ExpectApply(suite.expectedServer).Return(nil)
		mock2.ExpectApply(suite.expectedServer)
		mock3.ExpectApply(suite.expectedServer)

		suite.NoError(so.Apply(suite.expectedServer))

		mock0.AssertExpectations(suite.T())
		mock1.AssertExpectations(suite.T())
		mock2.AssertExpectations(suite.T())
		mock3.AssertExpectations(suite.T())
	})
}

func (suite *ServerOptionSuite) TestConnState() {
	var (
		called                = false
		expectedConn net.Conn = new(net.IPConn)
	)

	suite.Require().NoError(
		ConnState(func(actualConn net.Conn, cs http.ConnState) {
			suite.Same(expectedConn, actualConn)
			suite.Equal(http.StateNew, cs)
			called = true
		}).Apply(suite.expectedServer),
	)

	suite.expectedServer.ConnState(expectedConn, http.StateNew)
	suite.True(called)
}

func (suite *ServerOptionSuite) TestBaseContext() {
	type contextKey struct{}
	expectedCtx := context.WithValue(context.Background(), contextKey{}, "yes")

	suite.Require().NoError(
		BaseContext(func(net.Listener) context.Context {
			return expectedCtx
		}).Apply(suite.expectedServer),
	)

	suite.Require().NotNil(suite.expectedServer.BaseContext)
	suite.Same(
		expectedCtx,
		suite.expectedServer.BaseContext(nil),
	)
}

func (suite *ServerOptionSuite) TestConnContext() {
	type baseKey struct{}
	type connKey struct{}

	var (
		baseCtx = context.WithValue(context.Background(), baseKey{}, "yes")
		connCtx = context.WithValue(baseCtx, connKey{}, "yes")
	)

	suite.Require().NoError(
		ConnContext(func(ctx context.Context, _ net.Conn) context.Context {
			suite.Same(baseCtx, ctx)
			return connCtx
		}).Apply(suite.expectedServer),
	)

	suite.Require().NotNil(suite.expectedServer.ConnContext)
	suite.Same(
		connCtx,
		suite.expectedServer.ConnContext(baseCtx, nil),
	)
}

func (suite *ServerOptionSuite) TestErrorLog() {
	var (
		output   bytes.Buffer
		errorLog = log.New(&output, "test", log.LstdFlags)
	)

	suite.Require().NoError(
		ErrorLog(errorLog).Apply(suite.expectedServer),
	)

	suite.Require().NotNil(suite.expectedServer.ErrorLog)
	suite.expectedServer.ErrorLog.Printf("an error")
	suite.NotEmpty(output.String())
}

func TestServerOption(t *testing.T) {
	suite.Run(t, new(ServerOptionSuite))
}
