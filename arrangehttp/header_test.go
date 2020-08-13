package arrangehttp

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHeader(f func() Header, expected http.Header, t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		actual = make(http.Header)
		header Header

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

	assert.Equal(len(expected), header.Len())
	header.AddTo(actual)
	assert.Equal(expected, actual)

	request.Header.Set("Test", "true")
	decorated := header.AddResponse(handler)
	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(289, response.Code)
	assert.Equal(expected, response.HeaderMap)
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
			testHeader(
				func() Header { return NewHeader(record.src) },
				record.expected,
				t,
			)
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
			testHeader(
				func() Header { return NewHeaderFromMap(record.src) },
				record.expected,
				t,
			)
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
			testHeader(
				func() Header { return NewHeaders(record.src...) },
				record.expected,
				t,
			)
		})
	}
}
