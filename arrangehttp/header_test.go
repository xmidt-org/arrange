package arrangehttp

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHeaderBasic(f func() Header, expected http.Header, t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		header Header
		actual = make(http.Header)
	)

	require.NotPanics(func() {
		header = f()
	})

	assert.Equal(len(expected), header.Len())
	header.AddTo(actual)
	assert.Equal(expected, actual)
}

func testHeaderAddResponse(f func() Header, expected http.Header, t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		header   Header
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
		handler  = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Equal("true", request.Header.Get("Test"))
			response.WriteHeader(289)
		})
	)

	require.NotPanics(func() {
		header = f()
	})

	request.Header.Set("Test", "true")
	decorated := header.AddResponse(handler)
	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(289, response.Code)
	assert.Equal(expected, response.HeaderMap)
}

func testHeaderAddRequest(f func() Header, expected http.Header, t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		header  Header
		request = httptest.NewRequest("GET", "/", nil)

		roundTripper http.RoundTripper = RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
			if len(expected) == 0 {
				// allow for the nil Header case, since a nil won't compare equal to an http.Header{}
				assert.Empty(request.Header)
			} else {
				assert.Equal(expected, request.Header)
			}

			return &http.Response{
				StatusCode: 276,
			}, nil
		})
	)

	require.NotPanics(func() {
		header = f()
	})

	decorated := header.AddRequest(roundTripper)
	require.NotNil(decorated)
	response, err := decorated.RoundTrip(request)
	assert.NoError(err)
	require.NotNil(response)
	assert.Equal(276, response.StatusCode)

	// check that a nil Header still results in the round tripper creating headers
	request.Header = nil
	response, err = decorated.RoundTrip(request)
	assert.NoError(err)
	require.NotNil(response)
	assert.Equal(276, response.StatusCode)
}

func TestNewHeader(t *testing.T) {
	testData := []struct {
		src      http.Header
		expected http.Header
	}{
		{
			src:      nil,
			expected: http.Header{},
		},
		{
			src:      http.Header{},
			expected: http.Header{},
		},
		{
			src: http.Header{
				"content-type": {"text/plain"},
				"MultiVAlUe":   {"value1", "value2"},
				"blaNK":        {""},
				"emPTy":        {},
				"":             {"this shouldn't show up"},
			},
			expected: http.Header{
				"Content-Type": {"text/plain"},
				"Multivalue":   {"value1", "value2"},
				"Blank":        {""},
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f := func() Header { return NewHeader(record.src) }
			t.Run("Basic", func(t *testing.T) {
				testHeaderBasic(f, record.expected, t)
			})

			t.Run("AddResponse", func(t *testing.T) {
				testHeaderAddResponse(f, record.expected, t)
			})

			t.Run("AddRequest", func(t *testing.T) {
				testHeaderAddRequest(f, record.expected, t)
			})
		})
	}
}

func TestNewHeaderFromMap(t *testing.T) {
	testData := []struct {
		src      map[string]string
		expected http.Header
	}{
		{
			src:      nil,
			expected: http.Header{},
		},
		{
			src:      map[string]string{},
			expected: http.Header{},
		},
		{
			src: map[string]string{
				"content-type": "text/plain",
				"blaNK":        "",
				"":             "this shouldn't show up",
			},
			expected: http.Header{
				"Content-Type": {"text/plain"},
				"Blank":        {""},
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f := func() Header { return NewHeaderFromMap(record.src) }
			t.Run("Basic", func(t *testing.T) {
				testHeaderBasic(f, record.expected, t)
			})

			t.Run("AddResponse", func(t *testing.T) {
				testHeaderAddResponse(f, record.expected, t)
			})

			t.Run("AddRequest", func(t *testing.T) {
				testHeaderAddRequest(f, record.expected, t)
			})
		})
	}
}

func TestNewHeaders(t *testing.T) {
	testData := []struct {
		src      []string
		expected http.Header
	}{
		{
			src:      nil,
			expected: http.Header{},
		},
		{
			src:      []string{},
			expected: http.Header{},
		},
		{
			src: []string{
				"content-type", "text/plain",
				"blaNK", "",
				"multiValUE", "value1",
				"mUltivAlUE", "value2",
				"", "this shouldn't show up",
			},
			expected: http.Header{
				"Content-Type": {"text/plain"},
				"Multivalue":   {"value1", "value2"},
				"Blank":        {""},
			},
		},
		{
			src: []string{
				"content-type", "text/plain",
				"dangling",
			},
			expected: http.Header{
				"Content-Type": {"text/plain"},
				"Dangling":     {""},
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f := func() Header { return NewHeaders(record.src...) }
			t.Run("Basic", func(t *testing.T) {
				testHeaderBasic(f, record.expected, t)
			})

			t.Run("AddResponse", func(t *testing.T) {
				testHeaderAddResponse(f, record.expected, t)
			})

			t.Run("AddRequest", func(t *testing.T) {
				testHeaderAddRequest(f, record.expected, t)
			})
		})
	}
}
