package arrangehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange/arrangetls"
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
