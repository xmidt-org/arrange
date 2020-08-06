package arrangehttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRoundTripperChainNew(t *testing.T, testURL string) {
	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				callCount    int
				constructors []RoundTripperConstructor
			)

			for i := 0; i < length; i++ {
				constructors = append(constructors, func(next http.RoundTripper) http.RoundTripper {
					return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
						callCount++
						return next.RoundTrip(r)
					})
				})
			}

			chain := NewRoundTripperChain(constructors...)

			decorated := chain.Then(nil)
			require.NotNil(decorated) // should have used http.DefaultTransport
			response, err := decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
			require.NoError(err)
			require.NotNil(response)
			assert.Equal(299, response.StatusCode)
			assert.Equal(length, callCount)

			callCount = 0
			decorated = chain.Then(new(http.Transport))
			require.NotNil(decorated)
			response, err = decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
			require.NoError(err)
			require.NotNil(response)
			assert.Equal(299, response.StatusCode)
			assert.Equal(length, callCount)
		})
	}
}

func testRoundTripperChainAppend(t *testing.T, testURL string) {
	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				callCount    int
				constructors []RoundTripperConstructor
			)

			for i := 0; i < length; i++ {
				constructors = append(constructors, func(next http.RoundTripper) http.RoundTripper {
					return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
						callCount++
						return next.RoundTrip(r)
					})
				})
			}

			chain := NewRoundTripperChain().Append(constructors...)

			decorated := chain.Then(nil)
			require.NotNil(decorated) // should have used http.DefaultTransport
			response, err := decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
			require.NoError(err)
			require.NotNil(response)
			assert.Equal(299, response.StatusCode)
			assert.Equal(length, callCount)

			callCount = 0
			decorated = chain.Then(new(http.Transport))
			require.NotNil(decorated)
			response, err = decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
			require.NoError(err)
			require.NotNil(response)
			assert.Equal(299, response.StatusCode)
			assert.Equal(length, callCount)
		})
	}
}

func testRoundTripperChainExtend(t *testing.T, testURL string) {
	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				callCount    int
				constructors []RoundTripperConstructor
			)

			for i := 0; i < length; i++ {
				constructors = append(constructors, func(next http.RoundTripper) http.RoundTripper {
					return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
						callCount++
						return next.RoundTrip(r)
					})
				})
			}

			chain := NewRoundTripperChain().Extend(
				NewRoundTripperChain(constructors...),
			)

			decorated := chain.Then(nil)
			require.NotNil(decorated) // should have used http.DefaultTransport
			response, err := decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
			require.NoError(err)
			require.NotNil(response)
			assert.Equal(299, response.StatusCode)
			assert.Equal(length, callCount)

			callCount = 0
			decorated = chain.Then(new(http.Transport))
			require.NotNil(decorated)
			response, err = decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
			require.NoError(err)
			require.NotNil(response)
			assert.Equal(299, response.StatusCode)
			assert.Equal(length, callCount)
		})
	}
}

func testRoundTripperChainEmpty(t *testing.T, testURL string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	chain := NewRoundTripperChain()

	decorated := chain.Then(nil)
	assert.Nil(decorated)

	decorated = chain.Then(new(http.Transport))
	require.NotNil(decorated)
	response, err := decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(299, response.StatusCode)

	chain.Append()

	decorated = chain.Then(nil)
	assert.Nil(decorated)

	decorated = chain.Then(new(http.Transport))
	require.NotNil(decorated)
	response, err = decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(299, response.StatusCode)

	chain.Extend(NewRoundTripperChain())

	decorated = chain.Then(nil)
	assert.Nil(decorated)

	decorated = chain.Then(new(http.Transport))
	require.NotNil(decorated)
	response, err = decorated.RoundTrip(httptest.NewRequest("GET", testURL, nil))
	require.NoError(err)
	require.NotNil(response)
	assert.Equal(299, response.StatusCode)
}

func TestRoundTripperChain(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(299)
		}),
	)

	defer server.Close()

	t.Run("New", func(t *testing.T) { testRoundTripperChainNew(t, server.URL) })
	t.Run("Append", func(t *testing.T) { testRoundTripperChainAppend(t, server.URL) })
	t.Run("Extend", func(t *testing.T) { testRoundTripperChainExtend(t, server.URL) })
	t.Run("Empty", func(t *testing.T) { testRoundTripperChainEmpty(t, server.URL) })
}
