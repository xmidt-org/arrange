package arrangehttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	suite.ErrorAs(o.ApplyToServer(suite.expectedServer), &expectedErr)
	suite.Require().NotNil(expectedErr)
	suite.Require().NotNil(expectedErr.Type)
	suite.NotEmpty(expectedErr.Error())
}

func (suite *AsServerOptionSuite) TestTrivial() {
	expected := new(mockOption)
	suite.Same(expected, AsServerOption(expected))
	expected.AssertExpectations(suite.T())
}

func (suite *AsServerOptionSuite) TestApplyToServerNoError() {
	expected := new(mockOptionNoError)
	wrapper := AsServerOption(expected)
	suite.Require().NotNil(wrapper)

	expected.ExpectApplyToServer(suite.expectedServer)
	suite.NoError(wrapper.ApplyToServer(suite.expectedServer))
	expected.AssertExpectations(suite.T())
}

func (suite *AsServerOptionSuite) TestClosure() {
	expected := new(mockOption)
	wrapper := AsServerOption(expected.ApplyToServer)
	suite.Require().NotNil(wrapper)

	expected.ExpectApplyToServer(suite.expectedServer).Return(nil)
	suite.NoError(wrapper.ApplyToServer(suite.expectedServer))
	expected.AssertExpectations(suite.T())
}

func (suite *AsServerOptionSuite) TestClosureNoError() {
	expected := new(mockOptionNoError)
	wrapper := AsServerOption(expected.ApplyToServer)
	suite.Require().NotNil(wrapper)

	expected.ExpectApplyToServer(suite.expectedServer)
	suite.NoError(wrapper.ApplyToServer(suite.expectedServer))
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

func (suite *ServerOptionSuite) testServerOptionsEmpty() {
	suite.NoError(
		ServerOptions{}.ApplyToServer(suite.expectedServer),
	)
}

func (suite *ServerOptionSuite) testServerOptionsAllSuccess() {
	mocks := []*mockOption{
		new(mockOption),
		new(mockOption),
		new(mockOption),
	}

	mocks[0].ExpectApplyToServer(suite.expectedServer).Return(nil)
	mocks[1].ExpectApplyToServer(suite.expectedServer).Return(nil)
	mocks[2].ExpectApplyToServer(suite.expectedServer).Return(nil)

	suite.NoError(
		ServerOptions{
			mocks[0], mocks[1], mocks[2],
		}.ApplyToServer(suite.expectedServer),
	)

	mocks[0].AssertExpectations(suite.T())
	mocks[1].AssertExpectations(suite.T())
	mocks[2].AssertExpectations(suite.T())
}

func (suite *ServerOptionSuite) testServerOptionsAllFail() {
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

	mocks[0].ExpectApplyToServer(suite.expectedServer).Return(expectedErrors[0])
	mocks[1].ExpectApplyToServer(suite.expectedServer).Return(expectedErrors[1])
	mocks[2].ExpectApplyToServer(suite.expectedServer).Return(expectedErrors[2])

	actualErrors := multierr.Errors(
		ServerOptions{
			mocks[0], mocks[1], mocks[2],
		}.ApplyToServer(suite.expectedServer),
	)

	suite.Require().Len(actualErrors, len(expectedErrors))
	suite.Same(expectedErrors[0], actualErrors[0])
	suite.Same(expectedErrors[1], actualErrors[1])
	suite.Same(expectedErrors[2], actualErrors[2])

	mocks[0].AssertExpectations(suite.T())
	mocks[1].AssertExpectations(suite.T())
	mocks[2].AssertExpectations(suite.T())
}

func (suite *ServerOptionSuite) TestServerOptions() {
	suite.Run("Empty", suite.testServerOptionsEmpty)
	suite.Run("AllSuccess", suite.testServerOptionsAllSuccess)
	suite.Run("AllFail", suite.testServerOptionsAllFail)
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
		}).ApplyToServer(suite.expectedServer),
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
		).ApplyToServer(server),
	)

	suite.Require().NotNil(server.BaseContext)
	actualCtx := server.BaseContext(expectedListener)
	suite.Equal(expectedCtx, actualCtx)
}

func (suite *ServerOptionSuite) testConnContextNoInitial(count int) {
	type ctxKey struct{}
	expectedCtx := context.Background()

	s := &http.Server{
		ConnContext: nil, // start with no initial function
	}

	var fns []func(context.Context, net.Conn) context.Context
	for i := 0; i < count; i++ {
		i := i
		expectedCtx = context.WithValue(expectedCtx, ctxKey{}, strconv.Itoa(i))
		fns = append(fns, func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, ctxKey{}, strconv.Itoa(i))
		})
	}

	suite.NoError(
		ConnContext(fns...).ApplyToServer(s),
	)

	if count > 0 {
		suite.Require().NotNil(s.ConnContext)
		actualCtx := s.ConnContext(context.Background(), nil) // connection doesn't matter
		suite.Equal(expectedCtx, actualCtx)
	} else {
		suite.Nil(s.ConnContext)
	}
}

func (suite *ServerOptionSuite) testConnContextWithInitial(count int) {
	type ctxKey struct{}
	expectedCtx := context.WithValue(context.Background(), ctxKey{}, "initial")

	s := &http.Server{
		ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
			return context.WithValue(ctx, ctxKey{}, "initial")
		},
	}

	var fns []func(context.Context, net.Conn) context.Context
	for i := 0; i < count; i++ {
		i := i
		expectedCtx = context.WithValue(expectedCtx, ctxKey{}, strconv.Itoa(i))
		fns = append(fns, func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, ctxKey{}, strconv.Itoa(i))
		})
	}

	suite.NoError(
		ConnContext(fns...).ApplyToServer(s),
	)

	suite.Require().NotNil(s.ConnContext)
	actualCtx := s.ConnContext(context.Background(), nil) // connection doesn't matter
	suite.Equal(expectedCtx, actualCtx)
}

func (suite *ServerOptionSuite) TestConnContext() {
	suite.Run("NoInitial", func() {
		for _, count := range []int{0, 1, 2, 5} {
			suite.Run(fmt.Sprintf("count=%d", count), func() {
				suite.testConnContextNoInitial(count)
			})
		}
	})

	suite.Run("WithInitial", func() {
		for _, count := range []int{0, 1, 2, 5} {
			suite.Run(fmt.Sprintf("count=%d", count), func() {
				suite.testConnContextWithInitial(count)
			})
		}
	})
}

func (suite *ServerOptionSuite) TestErrorLog() {
	var (
		output   bytes.Buffer
		errorLog = log.New(&output, "test", log.LstdFlags)
	)

	suite.Require().NoError(
		ErrorLog(errorLog).ApplyToServer(suite.expectedServer),
	)

	suite.Require().NotNil(suite.expectedServer.ErrorLog)
	suite.expectedServer.ErrorLog.Printf("an error")
	suite.NotEmpty(output.String())
}

func (suite *ServerOptionSuite) testServerMiddleware(initialHandler http.Handler, count int) *httptest.ResponseRecorder {
	var (
		current    = 0
		middleware []func(http.Handler) http.Handler
		s          = &http.Server{
			Handler: initialHandler,
		}
	)

	for i := 0; i < count; i++ {
		i := i
		middleware = append(middleware, func(next http.Handler) http.Handler {
			suite.Require().NotNil(next)
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				suite.Equal(current, i)
				current++
				response.Header().Set(fmt.Sprintf("Middleware-%d", i), "true")
				next.ServeHTTP(response, request)
			})
		})
	}

	ServerMiddleware(middleware...).ApplyToServer(s)
	suite.Require().NotNil(s.Handler)

	var (
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	s.Handler.ServeHTTP(response, request)
	suite.Equal(count, current)
	for i := 0; i < count; i++ {
		suite.Equal(
			"true",
			response.HeaderMap.Get(fmt.Sprintf("Middleware-%d", i)),
		)
	}

	return response
}

func (suite *ServerOptionSuite) testServerMiddlewareNoHandler(count int) {
	response := suite.testServerMiddleware(nil, count)
	suite.Equal(404, response.Code) // uninitialized DefaultServeMux
}

func (suite *ServerOptionSuite) testServerMiddlewareWithHandler(count int) {
	response := suite.testServerMiddleware(
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.Header().Set("Handler", "true")
		}),
		count,
	)

	suite.Equal(
		"true",
		response.HeaderMap.Get("Handler"),
	)
}

func (suite *ServerOptionSuite) TestServerMiddleware() {
	suite.Run("NoHandler", func() {
		for _, count := range []int{0, 1, 2, 5} {
			suite.Run(fmt.Sprintf("count=%d", count), func() {
				suite.testServerMiddlewareNoHandler(count)
			})
		}
	})

	suite.Run("WithHandler", func() {
		for _, count := range []int{0, 1, 2, 5} {
			suite.Run(fmt.Sprintf("count=%d", count), func() {
				suite.testServerMiddlewareWithHandler(count)
			})
		}
	})
}

func TestServerOption(t *testing.T) {
	suite.Run(t, new(ServerOptionSuite))
}
