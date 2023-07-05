package arrangetest

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

// RequestMatcher is a Fluent Builder for a set of match criteria for an *http.Request.
// Used with mock.MatchedBy to match request object's by state rather than by identity.
type RequestMatcher struct {
	predicates []func(*http.Request) bool
}

// Match adds a predicate to this matcher, and returns this matcher for chaining.
func (rm *RequestMatcher) Match(p func(*http.Request) bool) *RequestMatcher {
	rm.predicates = append(rm.predicates, p)
	return rm
}

// Method matches on the request method.
func (rm *RequestMatcher) Method(v string) *RequestMatcher {
	return rm.Match(func(request *http.Request) bool {
		return request.Method == v
	})
}

// URL matches on the *http.Request.URL field.
func (rm *RequestMatcher) URL(v string) *RequestMatcher {
	return rm.Match(func(request *http.Request) bool {
		return request.URL != nil && request.URL.String() == v
	})
}

// Header matches on a request header.  For a multi-valued header,
// the expected value must appear in the actual list of values.
func (rm *RequestMatcher) Header(key, expected string) *RequestMatcher {
	return rm.Match(func(request *http.Request) bool {
		values := request.Header.Values(key)
		for _, v := range values {
			if v == expected {
				return true
			}
		}

		return false
	})
}

// Matches may be passed to mock.MatchedBy.  This method returns
// true if and only if all the predicates return true.
func (rm RequestMatcher) Matches(candidate *http.Request) (matched bool) {
	matched = true
	for i := 0; matched && i < len(rm.predicates); i++ {
		matched = matched && rm.predicates[i](candidate)
	}

	return
}

// RoundTripCall is a mocked Call that allows a clearer return declaration.
type RoundTripCall struct {
	*mock.Call
}

// Response sets the RoundTrip return to the given response with no error.
// The underlying *mock.Call is returned to continue method chaining if desired.
func (rtc RoundTripCall) Response(r *http.Response) *mock.Call {
	return rtc.Call.Return(r, error(nil))
}

// Error sets the RoundTrip return to the given error and a nil *http.Response.
// The underlying *mock.Call is returned to continue method chaining if desired.
func (rtc RoundTripCall) Error(err error) *mock.Call {
	return rtc.Call.Return((*http.Response)(nil), err)
}

// MockRoundTripper is a mocked http.RoundTripper.
type MockRoundTripper struct {
	mock.Mock
}

// RoundTrip executes the appropriate mocked call.
func (m *MockRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	args := m.Called(request)
	response, _ := args.Get(0).(*http.Response)
	return response, args.Error(1)
}

// Expect sets an expectation for the given request, returned a RoundTripCall
// to specify the return values and any other criteria.
func (m *MockRoundTripper) Expect(request *http.Request) RoundTripCall {
	return RoundTripCall{
		Call: m.On("RoundTrip", request),
	}
}

// ExpectMatch sets an expectation for a request matching the given criteria, and
// returns a RoundTripCall to specify return values and optionally other
// aspects of the call.
func (m *MockRoundTripper) ExpectMatch(matcher RequestMatcher) RoundTripCall {
	return RoundTripCall{
		Call: m.On("RoundTrip", mock.MatchedBy(matcher.Matches)),
	}
}
