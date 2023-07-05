package arrangetest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RequestMatcherSuite struct {
	suite.Suite
}

func parseURL(t require.TestingT, v string) *url.URL {
	u, err := url.Parse(v)
	require.NoError(t, err)
	return u
}

func (suite *RequestMatcherSuite) testMatchSuccess(count int) {
	called := 0
	expected := new(http.Request)
	var rm RequestMatcher
	for i := 0; i < count; i++ {
		rm.Match(func(actual *http.Request) bool {
			suite.Same(expected, actual)
			called++
			return true
		})
	}

	suite.True(rm.Matches(expected))
	suite.Equal(count, called)
}

func (suite *RequestMatcherSuite) testMatchFail(count int) {
	called := 0
	expected := new(http.Request)

	var rm RequestMatcher
	for i := 0; i < count-1; i++ {
		rm.Match(func(actual *http.Request) bool {
			suite.Same(expected, actual)
			called++
			return true
		})
	}

	rm.Match(func(actual *http.Request) bool {
		suite.Same(expected, actual)
		called++
		return false
	})

	suite.False(rm.Matches(expected))
	suite.Equal(count, called)
}

func (suite *RequestMatcherSuite) TestMatch() {
	suite.Run("Success", func() {
		for _, count := range []int{0, 1, 2, 5} {
			suite.Run(fmt.Sprintf("count-%d", count), func() {
				suite.testMatchSuccess(count)
			})
		}
	})

	suite.Run("Fail", func() {
		for _, count := range []int{1, 2, 5} {
			suite.Run(fmt.Sprintf("count-%d", count), func() {
				suite.testMatchFail(count)
			})
		}
	})
}

func (suite *RequestMatcherSuite) testURLMatch() {
	expected := &http.Request{
		URL: parseURL(suite.T(), "http://foobar.com/something"),
	}

	var rm RequestMatcher
	rm.URL("http://foobar.com/something")
	suite.True(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) testURLNoMatch() {
	expected := &http.Request{
		URL: parseURL(suite.T(), "http://foobar.com/something"),
	}

	var rm RequestMatcher
	rm.URL("http://foobar.com/mismatch")
	suite.False(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) TestURL() {
	suite.Run("Match", suite.testURLMatch)
	suite.Run("NoMatch", suite.testURLNoMatch)
}

func (suite *RequestMatcherSuite) testMethodMatch() {
	expected := &http.Request{
		Method: "XYZ",
	}

	var rm RequestMatcher
	rm.Method("XYZ")
	suite.True(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) testMethodNoMatch() {
	expected := &http.Request{
		Method: "XYZ",
	}

	var rm RequestMatcher
	rm.Method("GET")
	suite.False(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) TestMethod() {
	suite.Run("Match", suite.testMethodMatch)
	suite.Run("NoMatch", suite.testMethodNoMatch)
}

func (suite *RequestMatcherSuite) testHeaderSingleValueMatch() {
	expected := &http.Request{
		Header: http.Header{
			"Single": []string{"value"},
		},
	}

	var rm RequestMatcher
	rm.Header("Single", "value")
	suite.True(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) testHeaderSingleValueNoMatch() {
	expected := &http.Request{
		Header: http.Header{
			"Single": []string{"value"},
		},
	}

	var rm RequestMatcher
	rm.Header("Single", "nomatch")
	suite.False(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) testHeaderMultiValueMatch() {
	expected := &http.Request{
		Header: http.Header{
			"Multi": []string{"one", "two"},
		},
	}

	var rm RequestMatcher
	rm.Header("Multi", "two")
	suite.True(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) testHeaderMultiValueNoMatch() {
	expected := &http.Request{
		Header: http.Header{
			"Multi": []string{"one", "two"},
		},
	}

	var rm RequestMatcher
	rm.Header("Multi", "three")
	suite.False(rm.Matches(expected))
}

func (suite *RequestMatcherSuite) TestHeader() {
	suite.Run("SingleValue", func() {
		suite.Run("Match", suite.testHeaderSingleValueMatch)
		suite.Run("NoMatch", suite.testHeaderSingleValueNoMatch)
	})

	suite.Run("MultiValue", func() {
		suite.Run("Match", suite.testHeaderMultiValueMatch)
		suite.Run("NoMatch", suite.testHeaderMultiValueNoMatch)
	})
}

func TestRequestMatcher(t *testing.T) {
	suite.Run(t, new(RequestMatcherSuite))
}

type MockRoundTripperSuite struct {
	suite.Suite
}

func (suite *MockRoundTripperSuite) testExpectResponse() {
	var (
		expected = new(http.Response)
		request  = new(http.Request)
		rt       = new(MockRoundTripper)
	)

	rt.Expect(request).Response(expected).Once()
	actual, err := rt.RoundTrip(request)
	suite.NoError(err)
	suite.Same(expected, actual)
	rt.AssertExpectations(suite.T())
}

func (suite *MockRoundTripperSuite) testExpectError() {
	var (
		expected = errors.New("expected")
		request  = new(http.Request)
		rt       = new(MockRoundTripper)
	)

	rt.Expect(request).Error(expected).Once()
	response, actual := rt.RoundTrip(request)
	suite.Same(expected, actual)
	suite.Nil(response)
	rt.AssertExpectations(suite.T())
}

func (suite *MockRoundTripperSuite) TestExpect() {
	suite.Run("Response", suite.testExpectResponse)
	suite.Run("Error", suite.testExpectError)
}

func (suite *MockRoundTripperSuite) testExpectMatchResponse() {
	var (
		expected = new(http.Response)
		request  = &http.Request{
			Method: "GET",
			URL:    parseURL(suite.T(), "http://foo.com/query"),
			Header: http.Header{
				"Custom": []string{"value"},
			},
		}

		rm RequestMatcher
		rt = new(MockRoundTripper)
	)

	rm.Method("GET").
		URL("http://foo.com/query").
		Header("Custom", "value")

	rt.ExpectMatch(rm).Response(expected)

	actual, err := rt.RoundTrip(request)
	suite.NoError(err)
	suite.Same(expected, actual)
	rt.AssertExpectations(suite.T())
}

func (suite *MockRoundTripperSuite) testExpectMatchError() {
	var (
		expected = errors.New("expected")
		request  = &http.Request{
			Method: "GET",
			URL:    parseURL(suite.T(), "http://foo.com/query"),
			Header: http.Header{
				"Custom": []string{"value"},
			},
		}

		rm RequestMatcher
		rt = new(MockRoundTripper)
	)

	rm.Method("GET").
		URL("http://foo.com/query").
		Header("Custom", "value")

	rt.ExpectMatch(rm).Error(expected)

	response, actual := rt.RoundTrip(request)
	suite.Same(expected, actual)
	suite.Nil(response)
	rt.AssertExpectations(suite.T())
}

func (suite *MockRoundTripperSuite) TestExpectMatch() {
	suite.Run("Response", suite.testExpectMatchResponse)
	suite.Run("Error", suite.testExpectMatchError)
}

func TestMockRoundTripper(t *testing.T) {
	suite.Run(t, new(MockRoundTripperSuite))
}
