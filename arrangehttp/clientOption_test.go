package arrangehttp

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClientOptionsEmpty(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(ClientOptions()(nil))
}

func testClientOptionsSuccess(t *testing.T) {
	for _, count := range []int{0, 1, 2, 5} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var (
				assert = assert.New(t)

				expectedClient = &http.Client{
					Timeout: 125 * time.Minute,
				}

				options       []ClientOption
				expectedOrder []int
				actualOrder   []int
			)

			for i := 0; i < count; i++ {
				expectedOrder = append(expectedOrder, i)

				i := i
				options = append(options, func(actualClient *http.Client) error {
					assert.Equal(expectedClient, actualClient)
					actualOrder = append(actualOrder, i)
					return nil
				})
			}

			assert.NoError(
				ClientOptions(options...)(expectedClient),
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func testClientOptionsFailure(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedClient = &http.Client{
			Timeout: 45 * time.Second,
		}

		expectedErr = errors.New("expected")
		firstCalled bool

		co = ClientOptions(
			func(actualClient *http.Client) error {
				firstCalled = true
				assert.Equal(expectedClient, actualClient)
				return nil
			},
			func(actualClient *http.Client) error {
				assert.Equal(expectedClient, actualClient)
				return expectedErr
			},
			func(actualClient *http.Client) error {
				assert.Fail("This option should not have been called")
				return errors.New("This option should not have been called")
			},
		)
	)

	assert.Equal(
		expectedErr,
		co(expectedClient),
	)

	assert.True(firstCalled)
}

func TestClientOptions(t *testing.T) {
	t.Run("Empty", testClientOptionsEmpty)
	t.Run("Success", testClientOptionsSuccess)
	t.Run("Failure", testClientOptionsFailure)
}

func testNewCOptionUnsupported(t *testing.T) {
	assert := assert.New(t)
	assert.Nil(newCOption("unsupported type"))
}

func testNewCOptionSimple(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected = new(http.Client)

		literalCalled bool
		literal       = func(actual *http.Client) error {
			assert.True(expected == actual)
			literalCalled = true
			return nil
		}

		optionCalled bool
		option       ClientOption = func(actual *http.Client) error {
			optionCalled = true
			assert.True(expected == actual)
			return nil
		}
	)

	ci := &clientInfo{client: expected}
	co := newCOption(literal)
	require.NotNil(co)
	assert.NoError(co(ci))
	assert.True(literalCalled)

	ci = &clientInfo{client: expected}
	co = newCOption(option)
	require.NotNil(co)
	assert.NoError(co(ci))
	assert.True(optionCalled)
}

func testNewCOptionClientMiddlewareChain(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		chainCalled bool
		chain       = NewRoundTripperChain(
			func(next http.RoundTripper) http.RoundTripper {
				chainCalled = true
				return next
			},
		)

		ci = &clientInfo{client: new(http.Client)}
		co = newCOption(chain)
	)

	require.NotNil(co)
	assert.NoError(co(ci))
	ci.applyMiddleware()
	assert.NotNil(ci.client.Transport)
	assert.True(chainCalled)
}

func testNewCOptionConstructor(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		literalCalled bool
		literal       = func(next http.RoundTripper) http.RoundTripper {
			literalCalled = true
			return next
		}

		constructorCalled bool
		constructor       RoundTripperConstructor = func(next http.RoundTripper) http.RoundTripper {
			constructorCalled = true
			return next
		}
	)

	ci := &clientInfo{client: new(http.Client)}
	co := newCOption(literal)
	require.NotNil(co)
	assert.NoError(co(ci))
	ci.applyMiddleware()
	assert.NotNil(ci.client.Transport)
	assert.True(literalCalled)

	ci = &clientInfo{client: new(http.Client)}
	co = newCOption(constructor)
	require.NotNil(co)
	assert.NoError(co(ci))
	ci.applyMiddleware()
	assert.NotNil(ci.client.Transport)
	assert.True(constructorCalled)
}

func testNewCOptionConstructorSlice(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		literalsCalled []bool
		literals       = []func(http.RoundTripper) http.RoundTripper{
			func(next http.RoundTripper) http.RoundTripper {
				literalsCalled = append(literalsCalled, true)
				return next
			},
			func(next http.RoundTripper) http.RoundTripper {
				literalsCalled = append(literalsCalled, true)
				return next
			},
		}

		constructorsCalled []bool
		constructors       = []RoundTripperConstructor{
			func(next http.RoundTripper) http.RoundTripper {
				constructorsCalled = append(constructorsCalled, true)
				return next
			},
			func(next http.RoundTripper) http.RoundTripper {
				constructorsCalled = append(constructorsCalled, true)
				return next
			},
		}
	)

	ci := &clientInfo{client: new(http.Client)}
	co := newCOption(literals)
	require.NotNil(co)
	assert.NoError(co(ci))
	ci.applyMiddleware()
	assert.NotNil(ci.client.Transport)
	assert.Equal([]bool{true, true}, literalsCalled)

	ci = &clientInfo{client: new(http.Client)}
	co = newCOption(constructors)
	require.NotNil(co)
	assert.NoError(co(ci))
	ci.applyMiddleware()
	assert.NotNil(ci.client.Transport)
	assert.Equal([]bool{true, true}, constructorsCalled)
}

func TestNewCOption(t *testing.T) {
	t.Run("Unsupported", testNewCOptionUnsupported)
	t.Run("Simple", testNewCOptionSimple)
	t.Run("ClientMiddlewareChain", testNewCOptionClientMiddlewareChain)
	t.Run("Constructor", testNewCOptionConstructor)
	t.Run("ConstructorSlice", testNewCOptionConstructorSlice)
}
