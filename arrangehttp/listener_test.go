package arrangehttp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange/arrangetls"
)

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
