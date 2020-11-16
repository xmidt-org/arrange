package arrangehttp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange/arrangetls"
)

type simpleServerFactory struct {
	Address   string
	returnErr error
}

func (ssf simpleServerFactory) NewServer(http.Handler) (*http.Server, error) {
	if ssf.returnErr != nil {
		return nil, ssf.returnErr
	}

	return &http.Server{
		Addr: ssf.Address,
		// this factory does not set a handler, forcing the infrastructure to set it
	}, nil
}

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

	server, err := serverConfig.NewServer(router)
	require.NoError(err)
	require.NotNil(server)
	assert.Equal(router, server.Handler)

	assert.Equal(15*time.Second, server.ReadTimeout)
	assert.Equal(27*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(38*time.Second, server.WriteTimeout)
	assert.Equal(89*time.Minute, server.IdleTimeout)
	assert.Equal(478934, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
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

	server, err := serverConfig.NewServer(router)
	require.NoError(err)
	require.NotNil(server)
	assert.Equal(router, server.Handler)

	assert.Equal(72*time.Second, server.ReadTimeout)
	assert.Equal(109*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(63*time.Second, server.WriteTimeout)
	assert.Equal(9234*time.Minute, server.IdleTimeout)
	assert.Equal(3642, server.MaxHeaderBytes)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
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

func testServerConfigHeader(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverConfig = ServerConfig{
			Header: http.Header{
				"test1": {"true"},
				"test2": {"1", "2"},
			},
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	server, err := serverConfig.NewServer(router)
	require.NoError(err)
	require.NotNil(server)
	assert.NotNil(server.Handler)

	// check that this is a functioning server
	lf := NewListenerChain(CaptureListenAddress(address)).
		Factory(DefaultListenerFactory{})
	require.NoError(
		ServerOnStart(server, lf)(context.Background()),
	)

	defer server.Close()

	select {
	case listenAddress := <-address:
		response, err := http.Get("http://" + listenAddress.String())
		require.NoError(err)
		require.NotNil(response)
		defer response.Body.Close()
		io.Copy(ioutil.Discard, response.Body)

		assert.Equal([]string{"true"}, response.Header["Test1"])
		assert.Equal([]string{"1", "2"}, response.Header["Test2"])

	case <-time.After(2 * time.Second):
		assert.Fail("No captured listen address")
	}
}

func TestServerConfig(t *testing.T) {
	t.Run("Basic", testServerConfigBasic)
	t.Run("TLS", testServerConfigTLS)
	t.Run("Header", testServerConfigHeader)
}
