package arrangehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange/arrangetls"
	"go.uber.org/fx"
)

type testListenerDecorator struct {
	net.Listener
}

func (tld testListenerDecorator) Addr() net.Addr {
	return tld.Listener.Addr()
}

func testListenerChainNew(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				factory      DefaultListenerFactory
				constructors []ListenerConstructor
			)

			for i := 0; i < length; i++ {
				constructors = append(constructors, func(next net.Listener) net.Listener {
					return testListenerDecorator{Listener: next}
				})
			}

			listener, err := factory.Listen(context.Background(), &http.Server{Addr: ":0"})
			require.NoError(err)
			defer listener.Close()

			decorated := NewListenerChain(constructors...).Then(listener)
			require.NotNil(decorated)
			assert.NotNil(decorated.Addr())
		})
	}
}

func testListenerChainAppend(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				factory      DefaultListenerFactory
				constructors []ListenerConstructor
			)

			for i := 0; i < length; i++ {
				constructors = append(constructors, func(next net.Listener) net.Listener {
					return testListenerDecorator{Listener: next}
				})
			}

			listener, err := factory.Listen(context.Background(), &http.Server{Addr: ":0"})
			require.NoError(err)
			defer listener.Close()

			decorated := NewListenerChain().Append(constructors...).Then(listener)
			require.NotNil(decorated)
			assert.NotNil(decorated.Addr())
		})
	}
}

func testListenerChainExtend(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				factory      DefaultListenerFactory
				constructors []ListenerConstructor
			)

			for i := 0; i < length; i++ {
				constructors = append(constructors, func(next net.Listener) net.Listener {
					return testListenerDecorator{Listener: next}
				})
			}

			listener, err := factory.Listen(context.Background(), &http.Server{Addr: ":0"})
			require.NoError(err)
			defer listener.Close()

			decorated := NewListenerChain().Extend(NewListenerChain(constructors...)).Then(listener)
			require.NotNil(decorated)
			assert.NotNil(decorated.Addr())
		})
	}
}

func testListenerChainFactory(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				factory      DefaultListenerFactory
				constructors []ListenerConstructor
			)

			for i := 0; i < length; i++ {
				constructors = append(constructors, func(next net.Listener) net.Listener {
					return testListenerDecorator{Listener: next}
				})
			}

			decorated := NewListenerChain(constructors...).Factory(factory)
			require.NotNil(decorated)

			listener, err := decorated.Listen(context.Background(), &http.Server{Addr: ":0"})
			require.NoError(err)
			defer listener.Close()
			assert.NotNil(listener.Addr())
		})
	}
}

func TestListenerChain(t *testing.T) {
	t.Run("New", testListenerChainNew)
	t.Run("Append", testListenerChainAppend)
	t.Run("Extend", testListenerChainExtend)
	t.Run("Factory", testListenerChainFactory)
}

func testDefaultListenerFactoryBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		factory DefaultListenerFactory
		server  = &http.Server{
			Addr: ":0",
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	require.NoError(err)
	require.NotNil(listener)
	assert.NotNil(listener.Addr())
	listener.Close()
}

func testDefaultListenerFactoryTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		factory DefaultListenerFactory
		server  = &http.Server{
			Addr: ":0",
		}
	)

	tlsConfig, err := (&arrangetls.Config{
		Certificates: arrangetls.ExternalCertificates{
			{
				CertificateFile: CertificateFile,
				KeyFile:         KeyFile,
			},
		},
	}).New()

	require.NoError(err)
	require.NotNil(tlsConfig)
	server.TLSConfig = tlsConfig

	listener, err := factory.Listen(context.Background(), server)
	require.NoError(err)
	require.NotNil(listener)
	assert.NotNil(listener.Addr())
	listener.Close()
}

func testDefaultListenerFactoryListenError(t *testing.T) {
	var (
		assert = assert.New(t)

		factory = DefaultListenerFactory{
			Network: "this is a bad network",
		}

		server = &http.Server{
			Addr: ":0",
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	assert.Error(err)
	if !assert.Nil(listener) {
		// cleanup on a failed test
		listener.Close()
	}
}

func TestDefaultListenerFactory(t *testing.T) {
	t.Run("Basic", testDefaultListenerFactoryBasic)
	t.Run("TLS", testDefaultListenerFactoryTLS)
	t.Run("ListenError", testDefaultListenerFactoryListenError)
}

func TestCaptureListenAddress(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)
		chain   = NewListenerChain(CaptureListenAddress(address))

		server = &http.Server{
			Addr: ":0",
		}
	)

	listener, err := chain.Factory(DefaultListenerFactory{}).
		Listen(context.Background(), server)
	require.NoError(err)

	defer listener.Close()

	select {
	case listenAddr := <-address:
		assert.Equal(listener.Addr(), listenAddr)
	case <-time.After(2 * time.Second):
		assert.Fail("No listen address was sent to the channel")
	}
}

type testShutdowner struct {
	Called bool
}

func (ts *testShutdowner) Shutdown(...fx.ShutdownOption) error {
	ts.Called = true
	return nil
}

func TestShutdownOnExit(t *testing.T) {
	var (
		assert = assert.New(t)

		shutdowner = new(testShutdowner)
		serverExit = ShutdownOnExit(shutdowner)
	)

	serverExit()
	assert.True(shutdowner.Called)
}

func testServeNoServerExits(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		server = &http.Server{
			Addr: ":0",
		}

		result = make(chan error, 1)
	)

	listener, err := DefaultListenerFactory{}.Listen(context.Background(), server)
	require.NoError(err)
	defer listener.Close() // guard against a panic

	go func() {
		result <- Serve(server, listener)
	}()

	server.Close()

	select {
	case err := <-result:
		assert.Equal(http.ErrServerClosed, err)
	case <-time.After(2 * time.Second):
		assert.Fail("Serve failed to exit")
	}
}

func testServeWithServerExit(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		server = &http.Server{
			Addr: ":0",
		}

		result = make(chan error, 1)

		serverExitCalled = make(chan struct{})
		serverExit       = func() {
			close(serverExitCalled)
		}
	)

	listener, err := DefaultListenerFactory{}.Listen(context.Background(), server)
	require.NoError(err)
	defer listener.Close() // guard against a panic, as in a failed test

	go func() {
		result <- Serve(server, listener, serverExit)
	}()

	server.Close()

	select {
	case err := <-result:
		assert.Equal(http.ErrServerClosed, err)
	case <-time.After(2 * time.Second):
		assert.Fail("Serve failed to exit")
	}

	select {
	case <-serverExitCalled:
		// passing:
	case <-time.After(time.Second):
		assert.Fail("The server exit was not called")
	}
}

func TestServe(t *testing.T) {
	t.Run("NoServerExits", testServeNoServerExits)
	t.Run("WithServerExit", testServeWithServerExit)
}

func testServerOnStartNoServerExits(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		server = &http.Server{
			Addr: ":0",
		}

		result        = make(chan error, 1)
		serverOnStart = ServerOnStart(server, DefaultListenerFactory{})
	)

	require.NotNil(serverOnStart)
	go func() {
		result <- serverOnStart(context.Background())
	}()

	defer server.Close()

	select {
	case err := <-result:
		assert.Nil(err)
	case <-time.After(2 * time.Second):
		assert.Fail("The server on-start closure failed to exit")
	}
}

func testServerOnStartWithServerExit(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		server = &http.Server{
			Addr: ":0",
		}

		serverExitCalled = make(chan struct{})
		serverExit       = func() {
			close(serverExitCalled)
		}

		result        = make(chan error, 1)
		serverOnStart = ServerOnStart(server, DefaultListenerFactory{}, serverExit)
	)

	require.NotNil(serverOnStart)
	defer server.Close() // in case of a panic during a failed test
	go func() {
		result <- serverOnStart(context.Background())
	}()

	select {
	case err := <-result:
		assert.Nil(err)
	case <-time.After(2 * time.Second):
		assert.Fail("The server on-start closure failed to exit")
	}

	server.Close()

	select {
	case <-serverExitCalled:
		// passing
	case <-time.After(time.Second):
		assert.Fail("The server exit was not called")
	}
}

func testServerOnStartListenError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		server = &http.Server{
			Addr: ":0",
		}

		expectedErr = errors.New("expected error from Listen")

		result        = make(chan error, 1)
		serverOnStart = ServerOnStart(
			server,
			ListenerFactoryFunc(func(context.Context, *http.Server) (net.Listener, error) {
				return nil, expectedErr
			}),
		)
	)

	require.NotNil(serverOnStart)
	go func() {
		result <- serverOnStart(context.Background())
	}()

	defer server.Close()

	select {
	case err := <-result:
		assert.Equal(expectedErr, err)
	case <-time.After(2 * time.Second):
		assert.Fail("The server on-start closure failed to exit")
	}
}

func TestServerOnStart(t *testing.T) {
	t.Run("NoServerExits", testServerOnStartNoServerExits)
	t.Run("WithServerExit", testServerOnStartWithServerExit)
	t.Run("ListenError", testServerOnStartListenError)
}
