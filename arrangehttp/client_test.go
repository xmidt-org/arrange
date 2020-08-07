package arrangehttp

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func testTransportConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		tc = TransportConfig{
			TLSHandshakeTimeout:   15 * time.Second,
			DisableKeepAlives:     true,
			DisableCompression:    true,
			MaxIdleConns:          17,
			MaxIdleConnsPerHost:   5,
			MaxConnsPerHost:       92,
			IdleConnTimeout:       2 * time.Minute,
			ResponseHeaderTimeout: 13 * time.Millisecond,
			ExpectContinueTimeout: 29 * time.Second,
			ProxyConnectHeader: http.Header{
				"Something": []string{"Of Value"},
			},
			MaxResponseHeaderBytes: 347234,
			WriteBufferSize:        234867,
			ReadBufferSize:         93247,
			ForceAttemptHTTP2:      true,
		}
	)

	transport, err := tc.NewTransport(nil)
	require.NoError(err)
	require.NotNil(transport)

	assert.Nil(transport.TLSClientConfig)
	assert.Equal(15*time.Second, transport.TLSHandshakeTimeout)
	assert.True(transport.DisableKeepAlives)
	assert.True(transport.DisableCompression)
	assert.Equal(17, transport.MaxIdleConns)
	assert.Equal(5, transport.MaxIdleConnsPerHost)
	assert.Equal(92, transport.MaxConnsPerHost)
	assert.Equal(2*time.Minute, transport.IdleConnTimeout)
	assert.Equal(13*time.Millisecond, transport.ResponseHeaderTimeout)
	assert.Equal(29*time.Second, transport.ExpectContinueTimeout)
	assert.Equal(
		http.Header{"Something": []string{"Of Value"}},
		transport.ProxyConnectHeader,
	)
	assert.Equal(int64(347234), transport.MaxResponseHeaderBytes)
	assert.Equal(234867, transport.WriteBufferSize)
	assert.Equal(93247, transport.ReadBufferSize)
	assert.True(transport.ForceAttemptHTTP2)
}

func testTransportConfigTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		tc TransportConfig

		clientTLS = TLS{
			InsecureSkipVerify: true,
		}
	)

	transport, err := tc.NewTransport(&clientTLS)
	require.NoError(err)
	require.NotNil(transport)
	assert.NotNil(transport.TLSClientConfig)
}

func testTransportConfigError(t *testing.T) {
	var (
		assert = assert.New(t)

		tc TransportConfig

		clientTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: "missing",
					KeyFile:         "missing",
				},
			},
		}
	)

	transport, err := tc.NewTransport(&clientTLS)
	assert.Error(err)
	assert.NotNil(transport)
}

func TestTransportConfig(t *testing.T) {
	t.Run("Basic", testTransportConfigBasic)
	t.Run("TLS", testTransportConfigTLS)
	t.Run("Error", testTransportConfigError)
}

func testClientConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		cc = ClientConfig{
			Timeout: 15 * time.Second,
		}
	)

	client, err := cc.NewClient()
	require.NoError(err)
	require.NotNil(client)

	assert.Equal(15*time.Second, client.Timeout)
}

func testClientConfigError(t *testing.T) {
	var (
		assert = assert.New(t)

		cc = ClientConfig{
			TLS: &TLS{
				Certificates: ExternalCertificates{
					{
						CertificateFile: "missing",
						KeyFile:         "missing",
					},
				},
			},
		}
	)

	client, err := cc.NewClient()
	assert.Error(err)
	assert.NotNil(client)
}

func TestClientConfig(t *testing.T) {
	t.Run("Basic", testClientConfigBasic)
	t.Run("Error", testClientConfigError)
}

func testClientUnmarshal(t *testing.T, testURL string) {
	const yaml = `
timeout: "100s"
transport:
  disableCompression: true
  writeBufferSize: 4096
  readBufferSize: 4096
`
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v = viper.New()

		optionCalled = make(chan struct{})
		option       = func(client *http.Client) error {
			defer close(optionCalled)
			assert.Equal(100*time.Second, client.Timeout)
			return nil
		}

		client *http.Client
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
			Client(option).
				Use(func(next http.RoundTripper) http.RoundTripper {
					return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
						response, err := next.RoundTrip(request)
						if response != nil {
							response.Header.Set("Decorator-1", "value")
						}

						return response, err
					})
				}).
				UseChain(NewRoundTripperChain(func(next http.RoundTripper) http.RoundTripper {
					return RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
						response, err := next.RoundTrip(request)
						if response != nil {
							response.Header.Set("Decorator-2", "value")
						}

						return response, err
					})
				})).
				Unmarshal(),
		),
		fx.Populate(&client),
	)

	assert.NotNil(client)
	select {
	case <-optionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The client option was not called")
	}

	app.RequireStart()
	defer app.Stop(context.Background()) // in case of a failed test

	require.NotNil(client)
	request, err := http.NewRequest("GET", testURL, nil)
	require.NoError(err)
	require.NotNil(request)

	response, err := client.Do(request)
	require.NoError(err)
	require.NotNil(response)

	assert.Equal("value", response.Header.Get("Decorator-1"))
	assert.Equal("value", response.Header.Get("Decorator-2"))

	app.RequireStop()
}

func testClientUnmarshalError(t *testing.T) {
	const yaml = `
timeout: "this is not a valid golang duration"
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()
		client  *http.Client
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Client().Unmarshal(),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

type badClientFactory struct {
	Timeout time.Duration
}

func (bcf badClientFactory) NewClient() (*http.Client, error) {
	return nil, errors.New("expected client factory error")
}

func testClientFactoryError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()
		client *http.Client
	)

	v.Set("timeout", "89s")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Client().ClientFactory(badClientFactory{}).Unmarshal(),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

func testClientOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()
		client *http.Client
	)

	v.Set("timeout", "89s")
	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Client(func(*http.Client) error { return errors.New("expected option error") }).
				Unmarshal(),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

func testClientProvide(t *testing.T) {
	const yaml = `
timeout: "120ms"
transport:
  tlsHandshakeTimeout: "56s"
  disableCompression: false
`
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v = viper.New()

		optionCalled = make(chan struct{})
		option       = func(client *http.Client) error {
			defer close(optionCalled)
			assert.Equal(120*time.Millisecond, client.Timeout)
			return nil
		}

		client *http.Client
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		Client(option).Provide(),
		fx.Populate(&client),
	)

	assert.NotNil(client)
	select {
	case <-optionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The client option was not called")
	}

	// force the lifecycle to happen
	app.RequireStart()
	app.RequireStop()
}

func testClientUnmarshalKey(t *testing.T) {
	const yaml = `
clients:
  main:
    timeout: "25m"
    transport:
      disableCompression: true
`
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v = viper.New()

		optionCalled = make(chan struct{})
		option       = func(client *http.Client) error {
			defer close(optionCalled)
			assert.Equal(25*time.Minute, client.Timeout)
			return nil
		}

		client *http.Client
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
			Client(option).UnmarshalKey("clients.main"),
		),
		fx.Populate(&client),
	)

	assert.NotNil(client)
	select {
	case <-optionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The client option was not called")
	}

	// force the lifecycle to happen
	app.RequireStart()
	app.RequireStop()
}

func testClientUnmarshalKeyError(t *testing.T) {
	const yaml = `
clients:
  main:
    timeout: "this is not a valid golang duration"
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()
		client  *http.Client
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fx.New(
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		fx.Provide(
			Client().UnmarshalKey("clients.main"),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

func testClientProvideKey(t *testing.T) {
	const yaml = `
clients:
  main:
    timeout: "1716s"
    transport:
      forceAttemptHTTP2: true
`
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v = viper.New()

		optionCalled = make(chan struct{})
		option       = func(client *http.Client) error {
			defer close(optionCalled)
			assert.Equal(1716*time.Second, client.Timeout)
			return nil
		}

		client *http.Client
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	type ClientIn struct {
		fx.In
		Client *http.Client `name:"clients.main"`
	}

	app := fxtest.New(
		t,
		fx.Logger(
			log.New(ioutil.Discard, "", 0),
		),
		arrange.Supply(v),
		Client(option).ProvideKey("clients.main"),
		fx.Invoke(
			func(in ClientIn) {
				client = in.Client
			},
		),
	)

	assert.NotNil(client)
	select {
	case <-optionCalled:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("The client option was not called")
	}

	// force the lifecycle to happen
	app.RequireStart()
	app.RequireStop()
}

func TestClient(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(299)
		}),
	)

	defer server.Close()

	t.Run("Unmarshal", func(t *testing.T) { testClientUnmarshal(t, server.URL) })
	t.Run("UnmarshalError", testClientUnmarshalError)
	t.Run("FactoryError", testClientFactoryError)
	t.Run("OptionError", testClientOptionError)
	t.Run("Provide", testClientProvide)
	t.Run("UnmarshalKey", testClientUnmarshalKey)
	t.Run("UnmarshalKeyError", testClientUnmarshalKeyError)
	t.Run("ProvideKey", testClientProvideKey)
}
