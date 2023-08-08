package arrangehttp

import (
	"bytes"
	"context"
	"fmt"
	"github.com/xmidt-org/arrange/arrangetest"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServerOptionSuite struct {
	arrangetest.OptionSuite[http.Server]
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
		}).Apply(suite.Target),
	)

	suite.Target.ConnState(expectedConn, http.StateNew)
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
		ConnContext(fns...).Apply(s),
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
		ConnContext(fns...).Apply(s),
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
		ErrorLog(errorLog).Apply(suite.Target),
	)

	suite.Require().NotNil(suite.Target.ErrorLog)
	suite.Target.ErrorLog.Printf("an error")
	suite.NotEmpty(output.String())
}

func (suite *ServerOptionSuite) testServerMiddleware(initialHandler http.Handler) *httptest.ResponseRecorder {
	suite.Target.Handler = initialHandler

	ServerMiddleware(func(next http.Handler) http.Handler {
		suite.Require().NotNil(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.Header().Set("Middleware", "true")
			next.ServeHTTP(response, request)
		})
	}).Apply(suite.Target)

	suite.Require().NotNil(suite.Target.Handler)

	var (
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	suite.Target.Handler.ServeHTTP(response, request)
	suite.Equal(
		"true",
		response.Result().Header.Get("Middleware"),
	)

	return response
}

func (suite *ServerOptionSuite) testServerMiddlewareNoHandler() {
	response := suite.testServerMiddleware(nil)
	suite.Equal(404, response.Code) // uninitialized DefaultServeMux
}

func (suite *ServerOptionSuite) testServerMiddlewareWithHandler() {
	response := suite.testServerMiddleware(
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.Header().Set("Handler", "true")
		}),
	)

	suite.Equal(
		"true",
		response.Result().Header.Get("Handler"),
	)
}

func (suite *ServerOptionSuite) TestServerMiddleware() {
	suite.Run("NoHandler", suite.testServerMiddlewareNoHandler)
	suite.Run("WithHandler", suite.testServerMiddlewareWithHandler)
}

func TestServerOption(t *testing.T) {
	suite.Run(t, new(ServerOptionSuite))
}
