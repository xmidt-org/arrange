package arrangehttp

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
