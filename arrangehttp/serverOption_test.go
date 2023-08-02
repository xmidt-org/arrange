/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package arrangehttp

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServerOptionSuite struct {
	OptionSuite[http.Server]
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
		}).Apply(suite.target),
	)

	suite.target.ConnState(expectedConn, http.StateNew)
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
		ErrorLog(errorLog).Apply(suite.target),
	)

	suite.Require().NotNil(suite.target.ErrorLog)
	suite.target.ErrorLog.Printf("an error")
	suite.NotEmpty(output.String())
}

func (suite *ServerOptionSuite) testServerMiddleware(initialHandler http.Handler) *httptest.ResponseRecorder {
	suite.target.Handler = initialHandler

	ServerMiddleware(func(next http.Handler) http.Handler {
		suite.Require().NotNil(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.Header().Set("Middleware", "true")
			next.ServeHTTP(response, request)
		})
	}).Apply(suite.target)

	suite.Require().NotNil(suite.target.Handler)

	var (
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	suite.target.Handler.ServeHTTP(response, request)
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
