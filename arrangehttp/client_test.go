package arrangehttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetls"
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

		config = arrangetls.Config{
			InsecureSkipVerify: true,
		}
	)

	transport, err := tc.NewTransport(&config)
	require.NoError(err)
	require.NotNil(transport)
	assert.NotNil(transport.TLSClientConfig)
}

func testTransportConfigError(t *testing.T) {
	var (
		assert = assert.New(t)

		tc TransportConfig

		config = arrangetls.Config{
			Certificates: arrangetls.ExternalCertificates{
				{
					CertificateFile: "missing",
					KeyFile:         "missing",
				},
			},
		}
	)

	transport, err := tc.NewTransport(&config)
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
			TLS: &arrangetls.Config{
				Certificates: arrangetls.ExternalCertificates{
					{
						CertificateFile: "missing",
						KeyFile:         "missing",
					},
				},
			},
		}
	)

	_, err := cc.NewClient()
	assert.Error(err)
}

func TestClientConfig(t *testing.T) {
	t.Run("Basic", testClientConfigBasic)
	t.Run("Error", testClientConfigError)
}

func testCOptionsSuccess(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert    = assert.New(t)
				require   = require.New(t)
				client    = new(http.Client)
				options   []COption
				callCount int
			)

			for i := 0; i < length; i++ {
				options = append(options, func(c *http.Client) error {
					assert.Equal(client, c)
					callCount++
					return nil
				})
			}

			co := COptions(options...)
			require.NotNil(co)
			err := co(client)
			assert.NoError(err)
			assert.Equal(length, callCount)
		})
	}
}

func testCOptionsFailure(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		client      = new(http.Client)
		expectedErr = errors.New("expected option error")
		co          = COptions(
			func(c *http.Client) error {
				assert.Equal(client, c)
				return nil
			},
			func(c *http.Client) error {
				assert.Equal(client, c)
				return expectedErr
			},
			func(c *http.Client) error {
				assert.Fail("This option should not have been called")
				return errors.New("This option should not have been called")
			},
		)
	)

	require.NotNil(co)
	err := co(client)
	assert.Equal(expectedErr, err)
}

func TestCOptions(t *testing.T) {
	t.Run("Success", testCOptionsSuccess)
	t.Run("Failure", testCOptionsFailure)
}

func testNewCOptionUnsupported(t *testing.T) {
	assert := assert.New(t)
	co, err := NewCOption("this is not supported as an SOption")
	assert.Error(err)
	assert.Nil(co)
}

func testNewCOptionBasic(t *testing.T) {
	var (
		actualClient = new(*http.Client)
		optionErr    = errors.New("expected option error")
		testData     = []struct {
			option      interface{}
			expectedErr error
		}{
			{
				option: func(c *http.Client) error {
					*actualClient = c
					return nil
				},
			},
			{
				option: []func(*http.Client) error{
					func(c *http.Client) error {
						*actualClient = c
						return nil
					},
				},
			},
			{
				option: [1]func(*http.Client) error{
					func(c *http.Client) error {
						*actualClient = c
						return nil
					},
				},
			},
			{
				option: func(c *http.Client) error {
					*actualClient = c
					return optionErr
				},
				expectedErr: optionErr,
			},
			{
				option: func(c *http.Client) {
					*actualClient = c
				},
			},
			{
				option: []func(*http.Client){
					func(c *http.Client) {
						*actualClient = c
					},
				},
			},
			{
				option: [1]func(*http.Client){
					func(c *http.Client) {
						*actualClient = c
					},
				},
			},
		}
	)

	for i, record := range testData {
		*actualClient = nil
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				client  = new(http.Client)
				co, err = NewCOption(record.option)
			)

			require.NoError(err)
			require.NotNil(co)

			err = co(client)
			assert.Equal(record.expectedErr, err)
			assert.Equal(client, *actualClient)
		})
	}
}

func testNewCOptionRoundTripper(t *testing.T) {
	testData := []struct {
		option   interface{}
		expected http.Header
	}{
		{
			option: NewHeaders("Option", "true").AddRequest,
			expected: http.Header{
				"Option": {"true"},
			},
		},
		{
			option: RoundTripperConstructor(NewHeaders("Option", "true").AddRequest),
			expected: http.Header{
				"Option": {"true"},
			},
		},
		{
			option: []RoundTripperConstructor{
				NewHeaders("Option1", "true").AddRequest,
				NewHeaders("Option2", "true").AddRequest,
			},
			expected: http.Header{
				"Option1": {"true"},
				"Option2": {"true"},
			},
		},
		{
			option: [2]RoundTripperConstructor{
				NewHeaders("Option1", "true").AddRequest,
				NewHeaders("Option2", "true").AddRequest,
			},
			expected: http.Header{
				"Option1": {"true"},
				"Option2": {"true"},
			},
		},
		{
			option: NewRoundTripperChain(
				NewHeaders("Option1", "true").AddRequest,
				NewHeaders("Option2", "true").AddRequest,
			),
			expected: http.Header{
				"Option1": {"true"},
				"Option2": {"true"},
			},
		},
		{
			option: []RoundTripperChain{
				NewRoundTripperChain(
					NewHeaders("Option1", "true").AddRequest,
					NewHeaders("Option2", "true").AddRequest,
				),
				NewRoundTripperChain(
					NewHeaders("Option3", "true").AddRequest,
					NewHeaders("Option4", "true").AddRequest,
				),
			},
			expected: http.Header{
				"Option1": {"true"},
				"Option2": {"true"},
				"Option3": {"true"},
				"Option4": {"true"},
			},
		},
		{
			option: [2]RoundTripperChain{
				NewRoundTripperChain(
					NewHeaders("Option1", "true").AddRequest,
					NewHeaders("Option2", "true").AddRequest,
				),
				NewRoundTripperChain(
					NewHeaders("Option3", "true").AddRequest,
					NewHeaders("Option4", "true").AddRequest,
				),
			},
			expected: http.Header{
				"Option1": {"true"},
				"Option2": {"true"},
				"Option3": {"true"},
				"Option4": {"true"},
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				client  = new(http.Client)

				roundTripper http.RoundTripper = RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
					assert.Equal(record.expected, request.Header)
					return &http.Response{
						StatusCode: 876,
					}, nil
				})

				co, err = NewCOption(record.option)
			)

			require.NoError(err)
			require.NotNil(co)
			client.Transport = roundTripper
			require.NoError(co(client))

			request, err := http.NewRequest("GET", "/", nil)
			require.NoError(err)

			response, err := client.Do(request)
			require.NoError(err)
			require.NotNil(response)
			assert.Equal(876, response.StatusCode)
		})
	}
}

func TestNewCOption(t *testing.T) {
	t.Run("Unsupported", testNewCOptionUnsupported)
	t.Run("Basic", testNewCOptionBasic)
	t.Run("RoundTripper", testNewCOptionRoundTripper)
}

func testClientUnmarshal(t *testing.T) {
	type Dependencies struct {
		fx.In

		Global      RoundTripperChain
		NamedOption RoundTripperConstructor   `name:"test"`
		OptionGroup []RoundTripperConstructor `group:"client"`
	}

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

		client *http.Client
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	app := fxtest.New(
		t,
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			func() RoundTripperChain {
				return NewRoundTripperChain(
					NewHeaders("GlobalChain", "true").AddRequest,
				)
			},
			fx.Annotated{
				Name: "test",
				Target: func() RoundTripperConstructor {
					return NewHeaders("Named", "true").AddRequest
				},
			},
			fx.Annotated{
				Group: "client",
				Target: func() RoundTripperConstructor {
					return NewHeaders("Option1", "true").AddRequest
				},
			},
			fx.Annotated{
				Group: "client",
				Target: func() RoundTripperConstructor {
					return NewHeaders("Option2", "true").AddRequest
				},
			},
			Client().
				Use(
					NewHeaders("LocalConstructor", "true").AddRequest,
					NewRoundTripperChain(
						NewHeaders("LocalChain1", "true").AddRequest,
						NewHeaders("LocalChain2", "true").AddRequest,
					),
				).
				Inject(Dependencies{}).
				Unmarshal(),
		),
		fx.Populate(&client),
	)

	app.RequireStart()
	defer app.Stop(context.Background()) // in case of a failed test
	require.NotNil(client)

	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Equal("true", request.Header.Get("GlobalChain"))
			assert.Equal("true", request.Header.Get("Named"))
			assert.Equal("true", request.Header.Get("Option1"))
			assert.Equal("true", request.Header.Get("Option2"))
			assert.Equal("true", request.Header.Get("LocalConstructor"))
			assert.Equal("true", request.Header.Get("LocalChain1"))
			assert.Equal("true", request.Header.Get("LocalChain2"))
			response.WriteHeader(299)
		}),
	)

	defer server.Close()

	request, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(err)
	require.NotNil(request)

	response, err := client.Do(request)
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(299, response.StatusCode)

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
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			Client().Unmarshal(),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

func testClientUnmarshalUseError(t *testing.T) {
	const yaml = `
timeout: "90s"
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			Client().
				Use("this is not a supported option").
				Unmarshal(),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

func testClientUnmarshalInjectError(t *testing.T) {
	const yaml = `
timeout: "90s"
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			Client().
				Inject("this is not a struct that embeds fx.In").
				Unmarshal(),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

func testClientLocalCOptionError(t *testing.T) {
	var (
		assert = assert.New(t)
		client *http.Client

		v = viper.New()
	)

	v.Set("timeout", "10s")
	app := fx.New(
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			Client().
				Use(
					func(*http.Client) error { return errors.New("expected client option error") },
				).
				Unmarshal(),
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			Client().ClientFactory(badClientFactory{}).Unmarshal(),
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			Client().UnmarshalKey("clients.main"),
		),
		fx.Populate(&client),
	)

	assert.Error(app.Err())
}

func testClientUnmarshalKeyUseError(t *testing.T) {
	const yaml = `
clients:
  main:
    timeout: "90s"
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
		fx.Provide(
			Client().
				Use("this is not a supported option").
				UnmarshalKey("clients.main"),
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
		arrange.TestLogger(t),
		arrange.ForViper(v),
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
	t.Run("Unmarshal", testClientUnmarshal)
	t.Run("UnmarshalError", testClientUnmarshalError)
	t.Run("UnmarshalUseError", testClientUnmarshalUseError)
	t.Run("UnmarshalInjectError", testClientUnmarshalInjectError)
	t.Run("LocalCOptionError", testClientLocalCOptionError)
	t.Run("FactoryError", testClientFactoryError)
	t.Run("Provide", testClientProvide)
	t.Run("UnmarshalKey", testClientUnmarshalKey)
	t.Run("UnmarshalKeyError", testClientUnmarshalKeyError)
	t.Run("UnmarshalKeyUseError", testClientUnmarshalKeyUseError)
	t.Run("ProvideKey", testClientProvideKey)
}
