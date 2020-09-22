package arrangehttp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange/arrangetls"
)

func testServerConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverConfig = ServerConfig{
			Address:           ":0",
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 27 * time.Minute,
			WriteTimeout:      38 * time.Second,
			IdleTimeout:       89 * time.Minute,
			MaxHeaderBytes:    478934,
			KeepAlive:         16 * time.Minute,
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(server)

	assert.Equal(15*time.Second, server.ReadTimeout)
	assert.Equal(27*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(38*time.Second, server.WriteTimeout)
	assert.Equal(89*time.Minute, server.IdleTimeout)
	assert.Equal(478934, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	server.Handler = router
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		conn, err := net.Dial("tcp", listenAddress.String())
		require.NoError(err)
		defer conn.Close()

		fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
		_, err = bufio.NewReader(conn).ReadString('\n')
		require.NoError(err)

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func testServerConfigTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverConfig = ServerConfig{
			Address:           ":0",
			ReadTimeout:       72 * time.Second,
			ReadHeaderTimeout: 109 * time.Minute,
			WriteTimeout:      63 * time.Second,
			IdleTimeout:       9234 * time.Minute,
			MaxHeaderBytes:    3642,
			KeepAlive:         3 * time.Minute,
			TLS: &arrangetls.Config{
				Certificates: arrangetls.ExternalCertificates{
					{
						CertificateFile: CertificateFile,
						KeyFile:         KeyFile,
					},
				},
			},
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(server)

	assert.Equal(72*time.Second, server.ReadTimeout)
	assert.Equal(109*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(63*time.Second, server.WriteTimeout)
	assert.Equal(9234*time.Minute, server.IdleTimeout)
	assert.Equal(3642, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	server.Handler = router
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		conn, err := net.Dial("tcp", listenAddress.String())
		require.NoError(err)
		defer conn.Close()

		fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
		_, err = bufio.NewReader(conn).ReadString('\n')
		require.NoError(err)

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func TestServerConfig(t *testing.T) {
	t.Run("Basic", testServerConfigBasic)
	t.Run("TLS", testServerConfigTLS)
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
