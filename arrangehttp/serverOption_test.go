package arrangehttp

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestServerMiddlewareChain struct {
	handlers []func(http.Handler) http.Handler
}

func (tsmc TestServerMiddlewareChain) Then(next http.Handler) http.Handler {
	for i := len(tsmc.handlers) - 1; i >= 0; i-- {
		next = tsmc.handlers[i](next)
	}

	return next
}

var _ ServerMiddlewareChain = TestServerMiddlewareChain{}

type TestListener struct {
	R net.Conn
	W net.Conn
}

var _ net.Listener = (*TestListener)(nil)

func NewTestListener() *TestListener {
	tl := new(TestListener)
	tl.R, tl.W = net.Pipe()
	return tl
}

func (tl *TestListener) Accept() (net.Conn, error) {
	return tl.R, nil
}

func (tl *TestListener) Close() error {
	tl.R.Close()
	tl.W.Close()
	return nil
}

func (tl *TestListener) Addr() net.Addr {
	return tl.W.RemoteAddr()
}

func testServerOptionsEmpty(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(ServerOptions()(nil))
}

func testServerOptionsSuccess(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var (
				assert = assert.New(t)

				expectedServer = &http.Server{
					Addr: ":123",
				}

				options       []ServerOption
				expectedOrder []int
				actualOrder   []int
			)

			for i := 0; i < count; i++ {
				expectedOrder = append(expectedOrder, i)

				i := i
				options = append(options, func(actualServer *http.Server) error {
					assert.Equal(expectedServer, actualServer)
					actualOrder = append(actualOrder, i)
					return nil
				})
			}

			assert.NoError(
				ServerOptions(options...)(expectedServer),
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func testServerOptionsFailure(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedServer = &http.Server{
			Addr: ":456",
		}

		expectedErr = errors.New("expected")
		firstCalled bool

		so = ServerOptions(
			func(actualServer *http.Server) error {
				firstCalled = true
				assert.Equal(expectedServer, actualServer)
				return nil
			},
			func(actualServer *http.Server) error {
				assert.Equal(expectedServer, actualServer)
				return expectedErr
			},
			func(actualServer *http.Server) error {
				assert.Fail("This option should not have been called")
				return errors.New("This option should not have been called")
			},
		)
	)

	assert.Equal(
		expectedErr,
		so(expectedServer),
	)

	assert.True(firstCalled)
}

func TestServerOptions(t *testing.T) {
	t.Run("Empty", testServerOptionsEmpty)
	t.Run("Success", testServerOptionsSuccess)
	t.Run("Failure", testServerOptionsFailure)
}

func testRouterOptionsEmpty(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(RouterOptions()(nil))
}

func testRouterOptionsSuccess(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var (
				assert = assert.New(t)

				expectedRouter = mux.NewRouter()

				options       []RouterOption
				expectedOrder []int
				actualOrder   []int
			)

			for i := 0; i < count; i++ {
				expectedOrder = append(expectedOrder, i)

				i := i
				options = append(options, func(actualRouter *mux.Router) error {
					assert.Equal(expectedRouter, actualRouter)
					actualOrder = append(actualOrder, i)
					return nil
				})
			}

			assert.NoError(
				RouterOptions(options...)(expectedRouter),
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func testRouterOptionsFailure(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedRouter = mux.NewRouter()

		expectedErr = errors.New("expected")
		firstCalled bool

		ro = RouterOptions(
			func(actualRouter *mux.Router) error {
				firstCalled = true
				assert.Equal(expectedRouter, actualRouter)
				return nil
			},
			func(actualRouter *mux.Router) error {
				assert.Equal(expectedRouter, actualRouter)
				return expectedErr
			},
			func(actualRouter *mux.Router) error {
				assert.Fail("This option should not have been called")
				return errors.New("This option should not have been called")
			},
		)
	)

	assert.Equal(
		expectedErr,
		ro(expectedRouter),
	)

	assert.True(firstCalled)
}

func TestRouterOptions(t *testing.T) {
	t.Run("Empty", testRouterOptionsEmpty)
	t.Run("Success", testRouterOptionsSuccess)
	t.Run("Failure", testRouterOptionsFailure)
}

func testBaseContextNoBuilders(t *testing.T) {
	var (
		assert = assert.New(t)
		server http.Server
	)

	assert.NoError(BaseContext()(&server))
	assert.Nil(server.BaseContext)
}

func testBaseContextWithBuilders(t *testing.T) {
	for _, count := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			var (
				assert   = assert.New(t)
				require  = require.New(t)
				server   http.Server
				builders []func(context.Context, net.Listener) context.Context
			)

			for i := 0; i < count; i++ {
				i := i
				builders = append(builders, func(ctx context.Context, l net.Listener) context.Context {
					return context.WithValue(ctx, strconv.Itoa(i), l.Addr())
				})
			}

			require.NoError(BaseContext(builders...)(&server))
			require.NotNil(server.BaseContext)

			// make sure the test doesn't block without some kind of time limit
			listenCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			l, err := new(net.ListenConfig).Listen(listenCtx, "tcp", "")
			require.NoError(err)
			defer l.Close()

			actualCtx := server.BaseContext(l)
			for i := 0; i < count; i++ {
				// verify that each builder was called
				assert.Equal(
					l.Addr(),
					actualCtx.Value(strconv.Itoa(i)),
				)
			}
		})
	}
}

func TestBaseContext(t *testing.T) {
	t.Run("NoBuilders", testBaseContextNoBuilders)
	t.Run("WithBuilders", testBaseContextWithBuilders)
}

func testConnContextNoBuilders(t *testing.T) {
	var (
		assert = assert.New(t)
		server http.Server
	)

	assert.NoError(ConnContext()(&server))
	assert.Nil(server.ConnContext)
}

func testConnContextWithBuilders(t *testing.T) {
	for _, count := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			var (
				assert   = assert.New(t)
				require  = require.New(t)
				server   http.Server
				builders []func(context.Context, net.Conn) context.Context
			)

			for i := 0; i < count; i++ {
				i := i
				builders = append(builders, func(ctx context.Context, c net.Conn) context.Context {
					return context.WithValue(ctx, strconv.Itoa(i), c.LocalAddr())
				})
			}

			require.NoError(ConnContext(builders...)(&server))
			require.NotNil(server.ConnContext)

			conn, w := net.Pipe()
			defer conn.Close()
			defer w.Close()

			actualCtx := server.ConnContext(context.Background(), conn)
			for i := 0; i < count; i++ {
				// verify that each builder was called
				assert.Equal(
					conn.LocalAddr(),
					actualCtx.Value(strconv.Itoa(i)),
				)
			}
		})
	}
}

func TestConnContext(t *testing.T) {
	t.Run("NoBuilders", testConnContextNoBuilders)
	t.Run("WithBuilders", testConnContextWithBuilders)
}

func TestErrorLog(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = log.New(ioutil.Discard, "", 0)
		server http.Server
	)

	ErrorLog(nil)(&server)
	assert.Nil(server.ErrorLog)

	ErrorLog(logger)(&server)
	assert.Equal(logger, server.ErrorLog)

	ErrorLog(nil)(&server)
	assert.Nil(server.ErrorLog)
}

func testConnStateNoClosures(t *testing.T) {
	var (
		assert = assert.New(t)
		server http.Server
	)

	assert.NoError(ConnState()(&server))
	assert.Nil(server.ConnState)
}

func testConnStateWithClosures(t *testing.T) {
	type Result struct {
		Address net.Addr
		State   http.ConnState
	}

	for _, count := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			var (
				assert   = assert.New(t)
				require  = require.New(t)
				server   http.Server
				closures []func(net.Conn, http.ConnState)

				expected []Result
				actual   []Result
			)

			conn, w := net.Pipe()
			defer conn.Close()
			defer w.Close()

			for i := 0; i < count; i++ {
				expected = append(expected, Result{
					Address: conn.LocalAddr(),
					State:   http.StateHijacked,
				})

				closures = append(closures, func(c net.Conn, cs http.ConnState) {
					actual = append(actual, Result{
						Address: c.LocalAddr(),
						State:   cs,
					})
				})
			}

			require.NoError(ConnState(closures...)(&server))
			require.NotNil(server.ConnState)

			server.ConnState(conn, http.StateHijacked)

			// verify that each closure was called
			assert.Equal(expected, actual)
		})
	}
}

func TestConnState(t *testing.T) {
	t.Run("NoClosures", testConnStateNoClosures)
	t.Run("WithClosures", testConnStateWithClosures)
}

func testNewSOptionUnsupported(t *testing.T) {
	assert := assert.New(t)
	assert.Nil(newSOption("unsupported type"))
}

func testNewSOptionSimple(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected = new(http.Server)
		chain    ListenerChain

		literalCalled bool
		literal       = func(actual *http.Server) error {
			assert.True(expected == actual)
			literalCalled = true
			return nil
		}

		optionCalled bool
		option       ServerOption = func(actual *http.Server) error {
			assert.True(expected == actual)
			optionCalled = true
			return nil
		}
	)

	so := newSOption(literal)
	require.NotNil(so)
	lc, err := so(expected, nil, chain)
	assert.Equal(chain, lc)
	assert.NoError(err)
	assert.True(literalCalled)

	so = newSOption(option)
	require.NotNil(so)
	lc, err = so(expected, nil, chain)
	assert.Equal(chain, lc)
	assert.NoError(err)
	assert.True(optionCalled)
}

func testNewSOptionRouter(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected = new(mux.Router)
		chain    ListenerChain

		literalCalled bool
		literal       = func(actual *mux.Router) error {
			assert.True(expected == actual)
			literalCalled = true
			return nil
		}

		optionCalled bool
		option       RouterOption = func(actual *mux.Router) error {
			assert.True(expected == actual)
			optionCalled = true
			return nil
		}
	)

	so := newSOption(literal)
	require.NotNil(so)
	lc, err := so(nil, expected, chain)
	assert.Equal(chain, lc)
	assert.NoError(err)
	assert.True(literalCalled)

	so = newSOption(option)
	require.NotNil(so)
	lc, err = so(nil, expected, chain)
	assert.Equal(chain, lc)
	assert.NoError(err)
	assert.True(optionCalled)
}

func testNewSOptionMiddleware(t *testing.T) {
	type TestConstructor func(http.Handler) http.Handler

	var (
		assert  = assert.New(t)
		require = require.New(t)

		literal = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.Header().Set("Literal", "true")
				next.ServeHTTP(response, request)
			})
		}

		option TestConstructor = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.Header().Set("TestConstructor", "true")
				next.ServeHTTP(response, request)
			})
		}

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(267)
		})
	)

	so := newSOption(literal)
	require.NotNil(so)
	router := new(mux.Router)
	_, err := so(nil, router, ListenerChain{})
	assert.NoError(err)
	router.Handle("/", handler)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	router.ServeHTTP(response, request)
	assert.Equal(267, response.Code)
	assert.Equal("true", response.HeaderMap.Get("Literal"))

	so = newSOption(option)
	require.NotNil(so)
	router = new(mux.Router)
	_, err = so(nil, router, ListenerChain{})
	assert.NoError(err)
	router.Handle("/", handler)
	response = httptest.NewRecorder()
	request = httptest.NewRequest("GET", "/", nil)
	router.ServeHTTP(response, request)
	assert.Equal(267, response.Code)
	assert.Equal("true", response.HeaderMap.Get("TestConstructor"))
}

func testNewSOptionMiddlewareSlice(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		literal = []func(next http.Handler) http.Handler{
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					response.Header().Set("Literal-1", "true")
					next.ServeHTTP(response, request)
				})
			},
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					response.Header().Set("Literal-2", "true")
					next.ServeHTTP(response, request)
				})
			},
		}

		option = []mux.MiddlewareFunc{
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					response.Header().Set("TestConstructor-1", "true")
					next.ServeHTTP(response, request)
				})
			},
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					response.Header().Set("TestConstructor-2", "true")
					next.ServeHTTP(response, request)
				})
			},
		}

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(278)
		})
	)

	so := newSOption(literal)
	require.NotNil(so)
	router := new(mux.Router)
	_, err := so(nil, router, ListenerChain{})
	assert.NoError(err)
	router.Handle("/", handler)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	router.ServeHTTP(response, request)
	assert.Equal(278, response.Code)
	assert.Equal("true", response.HeaderMap.Get("Literal-1"))
	assert.Equal("true", response.HeaderMap.Get("Literal-2"))

	so = newSOption(option)
	require.NotNil(so)
	router = new(mux.Router)
	_, err = so(nil, router, ListenerChain{})
	assert.NoError(err)
	router.Handle("/", handler)
	response = httptest.NewRecorder()
	request = httptest.NewRequest("GET", "/", nil)
	router.ServeHTTP(response, request)
	assert.Equal(278, response.Code)
	assert.Equal("true", response.HeaderMap.Get("TestConstructor-1"))
	assert.Equal("true", response.HeaderMap.Get("TestConstructor-2"))
}

func testNewSOptionMiddlewareChain(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		router = new(mux.Router)

		chain = TestServerMiddlewareChain{
			handlers: []func(http.Handler) http.Handler{
				func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Header().Set("Chain-1", "true")
						next.ServeHTTP(response, request)
					})
				},
				func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						response.Header().Set("Chain-2", "true")
						next.ServeHTTP(response, request)
					})
				},
			},
		}

		handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(215)
		})

		so = newSOption(chain)
	)

	require.NotNil(so)
	_, err := so(nil, router, ListenerChain{})
	assert.NoError(err)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	router.Handle("/", handler)
	router.ServeHTTP(response, request)
	assert.Equal(215, response.Code)
	assert.Equal("true", response.HeaderMap.Get("Chain-1"))
	assert.Equal("true", response.HeaderMap.Get("Chain-2"))
}

func testNewSOptionListenerChain(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		listener    = NewTestListener()
		chainCalled []bool
		chain       = NewListenerChain(
			func(next net.Listener) net.Listener {
				chainCalled = append(chainCalled, true)
				return next
			},
			func(next net.Listener) net.Listener {
				chainCalled = append(chainCalled, true)
				return next
			},
		)

		so = newSOption(chain)
	)

	defer listener.Close()
	require.NotNil(so)
	lc, err := so(nil, nil, NewListenerChain())
	assert.NoError(err)
	decorated := lc.Then(listener)
	assert.Equal([]bool{true, true}, chainCalled)
	require.NotNil(decorated)
	c, err := decorated.Accept()
	assert.NoError(err)
	assert.Equal(listener.R, c)
}

func testNewSOptionListenerConstructor(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		listener = NewTestListener()

		literalCalled bool
		literal       = func(next net.Listener) net.Listener {
			literalCalled = true
			return next
		}

		optionCalled bool
		option       ListenerConstructor = func(next net.Listener) net.Listener {
			optionCalled = true
			return next
		}
	)

	defer listener.Close()

	so := newSOption(literal)
	lc, err := so(nil, nil, NewListenerChain())
	assert.NoError(err)
	decorated := lc.Then(listener)
	assert.True(literalCalled)
	require.NotNil(decorated)
	c, err := decorated.Accept()
	assert.NoError(err)
	assert.Equal(listener.R, c)

	so = newSOption(option)
	lc, err = so(nil, nil, NewListenerChain())
	assert.NoError(err)
	decorated = lc.Then(listener)
	assert.True(optionCalled)
	require.NotNil(decorated)
	c, err = decorated.Accept()
	assert.NoError(err)
	assert.Equal(listener.R, c)
}

func testNewSOptionListenerConstructorSlice(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		listener = NewTestListener()

		literalCalled []bool
		literal       = []func(net.Listener) net.Listener{
			func(next net.Listener) net.Listener {
				literalCalled = append(literalCalled, true)
				return next
			},
			func(next net.Listener) net.Listener {
				literalCalled = append(literalCalled, true)
				return next
			},
		}

		optionCalled []bool
		option       = []ListenerConstructor{
			func(next net.Listener) net.Listener {
				optionCalled = append(optionCalled, true)
				return next
			},
			func(next net.Listener) net.Listener {
				optionCalled = append(optionCalled, true)
				return next
			},
		}
	)

	defer listener.Close()

	so := newSOption(literal)
	lc, err := so(nil, nil, NewListenerChain())
	assert.NoError(err)
	decorated := lc.Then(listener)
	assert.Equal([]bool{true, true}, literalCalled)
	require.NotNil(decorated)
	c, err := decorated.Accept()
	assert.NoError(err)
	assert.Equal(listener.R, c)

	so = newSOption(option)
	lc, err = so(nil, nil, NewListenerChain())
	assert.NoError(err)
	decorated = lc.Then(listener)
	assert.Equal([]bool{true, true}, optionCalled)
	require.NotNil(decorated)
	c, err = decorated.Accept()
	assert.NoError(err)
	assert.Equal(listener.R, c)
}

func TestNewSOption(t *testing.T) {
	t.Run("Unsupported", testNewSOptionUnsupported)
	t.Run("Simple", testNewSOptionSimple)
	t.Run("Router", testNewSOptionRouter)
	t.Run("Middleware", testNewSOptionMiddleware)
	t.Run("MiddlewareSlice", testNewSOptionMiddlewareSlice)
	t.Run("MiddlewareChain", testNewSOptionMiddlewareChain)
	t.Run("ListenerChain", testNewSOptionListenerChain)
	t.Run("ListenerConstructor", testNewSOptionListenerConstructor)
	t.Run("ListenerConstructorSlice", testNewSOptionListenerConstructorSlice)
}
