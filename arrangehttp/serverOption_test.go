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
	expectedListener := new(net.TCPListener)
	type contextKey struct{}
	expectedCtx := context.WithValue(
		context.WithValue(context.Background(), contextKey{}, "0"),
		contextKey{}, "1",
	)

	server := new(http.Server)
	suite.Require().NoError(
		BaseContext(
			func(ctx context.Context, actualListener net.Listener) context.Context {
				suite.Same(expectedListener, actualListener)
				return context.WithValue(ctx, contextKey{}, "0")
			},
			func(ctx context.Context, actualListener net.Listener) context.Context {
				suite.Same(expectedListener, actualListener)
				return context.WithValue(ctx, contextKey{}, "1")
			},
		).Apply(server),
	)

	suite.Require().NotNil(server.BaseContext)
	actualCtx := server.BaseContext(expectedListener)
	suite.Equal(expectedCtx, actualCtx)
}

func (suite *ServerOptionSuite) testConnContextSimple() {
	type baseKey struct{}
	type connKey struct{}

	var (
		baseCtx     = context.WithValue(context.Background(), baseKey{}, "base")
		expectedCtx = context.WithValue(baseCtx, connKey{}, "conn")
	)

	suite.Require().NoError(
		ConnContext(func(ctx context.Context, _ net.Conn) context.Context {
			return context.WithValue(ctx, connKey{}, "conn")
		}).Apply(suite.expectedServer),
	)

	suite.Require().NotNil(suite.expectedServer.ConnContext)
	suite.Equal(
		expectedCtx,
		suite.expectedServer.ConnContext(baseCtx, nil),
	)
}

func (suite *ServerOptionSuite) testConnContextExisting() {
	type existingKey struct{}
	type baseKey struct{}
	type connKey1 struct{}
	type connKey2 struct{}

	var (
		baseCtx     = context.WithValue(context.Background(), baseKey{}, "base")
		expectedCtx = context.WithValue(
			context.WithValue(
				context.WithValue(baseCtx, existingKey{}, "existing"),
				connKey1{},
				"conn1",
			),
			connKey2{},
			"conn2",
		)
	)

	suite.expectedServer.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, existingKey{}, "existing")
	}

	suite.Require().NoError(
		ConnContext(
			func(ctx context.Context, _ net.Conn) context.Context {
				return context.WithValue(ctx, connKey1{}, "conn1")
			},
			func(ctx context.Context, _ net.Conn) context.Context {
				return context.WithValue(ctx, connKey2{}, "conn2")
			},
		).Apply(suite.expectedServer),
	)

	suite.Require().NotNil(suite.expectedServer.ConnContext)
	suite.Equal(
		expectedCtx,
		suite.expectedServer.ConnContext(baseCtx, nil),
	)
}

func (suite *ServerOptionSuite) TestConnContext() {
	suite.Run("Simple", suite.testConnContextSimple)
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

func (suite *ServerOptionSuite) testServerMiddlewareNoHandler() {
	var (
		s = new(http.Server)
	)

	ServerOptions{
		ServerMiddleware(
			func(h http.Handler) http.Handler {
				return h
			},
			func(h http.Handler) http.Handler {
				return h
			},
		),
	}.Apply(s)

	suite.Same(
		http.DefaultServeMux,
		s.Handler,
	)
}

func (suite *ServerOptionSuite) testServerMiddlewareWithHandler() {
	var (
		expected = new(http.ServeMux)

		s = &http.Server{
			Handler: expected,
		}
	)

	ServerOptions{
		ServerMiddleware(
			func(h http.Handler) http.Handler {
				return h
			},
			func(h http.Handler) http.Handler {
				return h
			},
		),
	}.Apply(s)

	suite.Same(expected, s.Handler)
}

func (suite *ServerOptionSuite) TestServerMiddleware() {
	suite.Run("NoHandler", suite.testServerMiddlewareNoHandler)
	suite.Run("WithHandler", suite.testServerMiddlewareWithHandler)
}

func TestServerOption(t *testing.T) {
	suite.Run(t, new(ServerOptionSuite))
}
