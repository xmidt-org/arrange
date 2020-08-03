package arrangehttp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
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

	server, listen, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(listen)
	require.NotNil(server)

	assert.Equal(15*time.Second, server.ReadTimeout)
	assert.Equal(27*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(38*time.Second, server.WriteTimeout)
	assert.Equal(89*time.Minute, server.IdleTimeout)
	assert.Equal(478934, server.MaxHeaderBytes)

	// check that this is a functioning server
	listen = NewListenerChain(CaptureListenAddress(address)).Listen(listen)
	server.Handler = router
	require.NoError(
		ServerOnStart(server, listen)(context.Background()),
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

		certificateFile, keyFile = createServerFiles(t)

		serverConfig = ServerConfig{
			Address:           ":0",
			ReadTimeout:       72 * time.Second,
			ReadHeaderTimeout: 109 * time.Minute,
			WriteTimeout:      63 * time.Second,
			IdleTimeout:       9234 * time.Minute,
			MaxHeaderBytes:    3642,
			KeepAlive:         3 * time.Minute,
			TLS: &ServerTLS{
				CertificateFile: certificateFile,
				KeyFile:         keyFile,
			},
		}

		router  = mux.NewRouter()
		address = make(chan net.Addr, 1)
	)

	defer os.Remove(certificateFile)
	defer os.Remove(keyFile)

	server, listen, err := serverConfig.NewServer()
	require.NoError(err)
	require.NotNil(listen)
	require.NotNil(server)

	assert.Equal(72*time.Second, server.ReadTimeout)
	assert.Equal(109*time.Minute, server.ReadHeaderTimeout)
	assert.Equal(63*time.Second, server.WriteTimeout)
	assert.Equal(9234*time.Minute, server.IdleTimeout)
	assert.Equal(3642, server.MaxHeaderBytes)

	// check that this is a functioning server
	listen = NewListenerChain(CaptureListenAddress(address)).Listen(listen)
	server.Handler = router
	require.NoError(
		ServerOnStart(server, listen)(context.Background()),
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

func testServerListenerConstructors(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		address1 = make(chan net.Addr, 1)
		address2 = make(chan net.Addr, 1)

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().
				AppendListenerConstructors(
					CaptureListenAddress(address1),
				).
				ExtendListenerConstructors(
					NewListenerChain(CaptureListenAddress(address2)),
				).
				Unmarshal(),
		),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	var serverAddress net.Addr
	select {
	case serverAddress = <-address1:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	select {
	case secondAddress := <-address2:
		assert.Equal(serverAddress, secondAddress)
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func testServerUnmarshal(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		serverOptionCalled = make(chan struct{})
		serverOption       = func(s *http.Server) error {
			defer close(serverOptionCalled)
			assert.NotNil(s)
			return nil
		}

		routerOptionCalled = make(chan struct{})
		routerOption       = func(r *mux.Router) error {
			defer close(routerOptionCalled)
			assert.NotNil(r)
			return nil
		}

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server(serverOption).
				RouterOptions(routerOption).
				AppendListenerConstructors(CaptureListenAddress(address)).
				Unmarshal(),
		),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-serverOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The server option was not called")
	}

	select {
	case <-routerOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The router option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

type badServerFactory struct {
	Address string
}

func (bsf badServerFactory) NewServer() (*http.Server, Listen, error) {
	return nil, nil, errors.New("factory error")
}

func testServerServerFactoryError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", "localhost:8080")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().
				ServerFactory(badServerFactory{}).
				Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerServerOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", "localhost:8080")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server(func(*http.Server) error { return errors.New("expected server option error") }).
				Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerRouterOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", "localhost:8080")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().
				RouterOptions(func(*mux.Router) error { return errors.New("expected router option error") }).
				Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerUnmarshalError(t *testing.T) {
	var (
		assert = assert.New(t)
		router *mux.Router

		v = viper.New()
	)

	v.Set("address", ":0")
	v.Set("readTimeout", "this is not a valid golang time.Duration")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().Unmarshal(),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerProvide(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		serverOptionCalled = make(chan struct{})
		serverOption       = func(s *http.Server) error {
			defer close(serverOptionCalled)
			assert.NotNil(s)
			return nil
		}

		routerOptionCalled = make(chan struct{})
		routerOption       = func(r *mux.Router) error {
			defer close(routerOptionCalled)
			assert.NotNil(r)
			return nil
		}

		v = viper.New()
	)

	v.Set("address", ":0")
	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		Server(serverOption).
			RouterOptions(routerOption).
			AppendListenerConstructors(CaptureListenAddress(address)).
			Provide(),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-serverOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The server option was not called")
	}

	select {
	case <-routerOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The router option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func testServerUnmarshalKey(t *testing.T) {
	const yaml = `
servers:
  main:
    address: ":0"
    readTimeout: "30s"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		serverOptionCalled = make(chan struct{})
		serverOption       = func(s *http.Server) error {
			defer close(serverOptionCalled)
			if assert.NotNil(s) {
				assert.Equal(30*time.Second, s.ReadTimeout)
			}

			return nil
		}

		routerOptionCalled = make(chan struct{})
		routerOption       = func(r *mux.Router) error {
			defer close(routerOptionCalled)
			assert.NotNil(r)
			return nil
		}

		v = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server(serverOption).
				RouterOptions(routerOption).
				AppendListenerConstructors(CaptureListenAddress(address)).
				UnmarshalKey("servers.main"),
		),
		fx.Invoke(
			func(r *mux.Router) {
				r.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-serverOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The server option was not called")
	}

	select {
	case <-routerOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The router option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func testServerUnmarshalKeyError(t *testing.T) {
	const yaml = `
servers:
  main:
    address: ":0"
    readTimeout: "this is not a valid golang time.Duration"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		router  *mux.Router

		v = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Server().UnmarshalKey("servers.main"),
		),
		fx.Populate(&router),
	)

	assert.Error(app.Err())
}

func testServerProvideKey(t *testing.T) {
	const yaml = `
servers:
  main:
    address: ":0"
    readTimeout: "30s"
`

	var (
		assert  = assert.New(t)
		require = require.New(t)

		address = make(chan net.Addr, 1)

		serverOptionCalled = make(chan struct{})
		serverOption       = func(s *http.Server) error {
			defer close(serverOptionCalled)
			if assert.NotNil(s) {
				assert.Equal(30*time.Second, s.ReadTimeout)
			}

			return nil
		}

		routerOptionCalled = make(chan struct{})
		routerOption       = func(r *mux.Router) error {
			defer close(routerOptionCalled)
			assert.NotNil(r)
			return nil
		}

		v = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	type RouterIn struct {
		fx.In
		Router *mux.Router `name:"servers.main"`
	}

	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		Server(serverOption).
			RouterOptions(routerOption).
			AppendListenerConstructors(CaptureListenAddress(address)).
			ProvideKey("servers.main"),
		fx.Invoke(
			func(r RouterIn) {
				r.Router.HandleFunc("/test", func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(277)
				})
			},
		),
	)

	app.RequireStart()
	defer app.Stop(context.Background())

	select {
	case <-serverOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The server option was not called")
	}

	select {
	case <-routerOptionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The router option was not called")
	}

	var serverAddress net.Addr
	select {
	case serverAddress = <-address:
	case <-time.After(2 * time.Second):
		assert.Fail("No server address returned")
	}

	response, err := http.Get("http://" + serverAddress.String() + "/test")
	require.NoError(err)
	assert.Equal(277, response.StatusCode)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}

func TestServer(t *testing.T) {
	t.Run("ListenerConstructors", testServerListenerConstructors)
	t.Run("Unmarshal", testServerUnmarshal)
	t.Run("UnmarshalError", testServerUnmarshalError)
	t.Run("FactoryError", testServerServerFactoryError)
	t.Run("ServerOptionError", testServerServerOptionError)
	t.Run("RouterOptionError", testServerRouterOptionError)
	t.Run("Provide", testServerProvide)
	t.Run("UnmarshalKey", testServerUnmarshalKey)
	t.Run("UnmarshalKeyError", testServerUnmarshalKeyError)
	t.Run("ProvideKey", testServerProvideKey)
}
