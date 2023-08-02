/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package arrangehttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetls"
	"github.com/xmidt-org/httpaux/httpmock"
)

type ClientConfigSuite struct {
	arrangetls.Suite
	server            *httptest.Server
	requestAssertions []func(*http.Request)
}

func (suite *ClientConfigSuite) handleTestRequest(response http.ResponseWriter, request *http.Request) {
	for _, ra := range suite.requestAssertions {
		ra(request)
	}

	response.WriteHeader(299)
}

func (suite *ClientConfigSuite) addTestRequestAssertions(ra ...func(*http.Request)) {
	suite.requestAssertions = append(suite.requestAssertions, ra...)
}

func (suite *ClientConfigSuite) SetupSuite() {
	suite.Suite.SetupSuite()
	suite.server = httptest.NewServer(
		http.HandlerFunc(suite.handleTestRequest),
	)
}

func (suite *ClientConfigSuite) TearDownTest() {
	suite.requestAssertions = nil
}

func (suite *ClientConfigSuite) TearDownSuite() {
	suite.Suite.TearDownSuite()
	suite.server.Close()
}

// expectedTransportConfig returns a TransportConfig with everything set to a distinct,
// non-default value.
func (suite *ClientConfigSuite) expectedTransportConfig() TransportConfig {
	return TransportConfig{
		TLSHandshakeTimeout:   20 * time.Second,
		DisableKeepAlives:     true,
		DisableCompression:    true,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       12,
		IdleConnTimeout:       13 * time.Minute,
		ResponseHeaderTimeout: 12 * time.Hour,
		ExpectContinueTimeout: 5 * time.Second,
		ProxyConnectHeader: http.Header{
			"Test": []string{"Value"},
		},
		MaxResponseHeaderBytes: 4096,
		WriteBufferSize:        1123,
		ReadBufferSize:         9473,
		ForceAttemptHTTP2:      true,
	}
}

// assertTransport asserts that an *http.Transport was correctly created from a TransportConfig.
func (suite *ClientConfigSuite) assertTransport(expected TransportConfig, actual *http.Transport) {
	suite.Equal(expected.TLSHandshakeTimeout, actual.TLSHandshakeTimeout)
	suite.Equal(expected.DisableKeepAlives, actual.DisableKeepAlives)
	suite.Equal(expected.DisableCompression, actual.DisableCompression)
	suite.Equal(expected.MaxIdleConns, actual.MaxIdleConns)
	suite.Equal(expected.MaxIdleConnsPerHost, actual.MaxIdleConnsPerHost)
	suite.Equal(expected.MaxConnsPerHost, actual.MaxConnsPerHost)
	suite.Equal(expected.IdleConnTimeout, actual.IdleConnTimeout)
	suite.Equal(expected.ResponseHeaderTimeout, actual.ResponseHeaderTimeout)
	suite.Equal(expected.ExpectContinueTimeout, actual.ExpectContinueTimeout)
	suite.Equal(expected.ProxyConnectHeader, actual.ProxyConnectHeader)
	suite.Equal(expected.MaxResponseHeaderBytes, actual.MaxResponseHeaderBytes)
	suite.Equal(expected.WriteBufferSize, actual.WriteBufferSize)
	suite.Equal(expected.ReadBufferSize, actual.ReadBufferSize)
	suite.Equal(expected.ForceAttemptHTTP2, actual.ForceAttemptHTTP2)
}

func (suite *ClientConfigSuite) assertClient(expected ClientConfig, actual *http.Client) {
	suite.Equal(expected.Timeout, actual.Timeout)
}

// getClient obtains an *http.Client, expecting no errors.  This method calls assertClient
// prior to returning.
func (suite *ClientConfigSuite) getClient(cc ClientConfig) *http.Client {
	c, err := cc.NewClient()
	suite.Require().NoError(err)
	suite.Require().NotNil(c)
	suite.assertClient(cc, c)
	return c
}

// sendRequest sends a request to the test server.  The response body is consumed and closed
// prior to returning.
func (suite *ClientConfigSuite) sendRequest(client *http.Client, method string, body io.Reader) *http.Response {
	request, err := http.NewRequest(method, suite.server.URL, body)
	suite.Require().NoError(err)
	suite.Require().NotNil(request)

	response, err := client.Do(request)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	return response
}

func (suite *ClientConfigSuite) testTransportConfigNoTLS() {
	var (
		expected    = suite.expectedTransportConfig()
		actual, err = expected.NewTransport(nil)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(actual)
	suite.assertTransport(expected, actual)
	suite.Nil(actual.TLSClientConfig)
}

func (suite *ClientConfigSuite) testTransportConfigTLS() {
	var (
		expected    = suite.expectedTransportConfig()
		actual, err = expected.NewTransport(suite.Config())
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(actual)
	suite.assertTransport(expected, actual)
	suite.Require().NotNil(actual.TLSClientConfig)
}

func (suite *ClientConfigSuite) TestTransportConfig() {
	suite.Run("NoTLS", suite.testTransportConfigNoTLS)
	suite.Run("TLS", suite.testTransportConfigTLS)
}

func (suite *ClientConfigSuite) TestNewClient() {
	cc := ClientConfig{
		Timeout: 15 * time.Second,
	}

	client := suite.getClient(cc)

	response := suite.sendRequest(client, "GET", nil)
	suite.Equal(299, response.StatusCode)
}

func (suite *ClientConfigSuite) testApplyNoHeader() {
	cc := ClientConfig{
		Timeout: 15 * time.Second,
	}

	client := new(http.Client)
	suite.Require().NoError(cc.Apply(client))

	response := suite.sendRequest(client, "GET", nil)
	suite.Equal(299, response.StatusCode)
}

func (suite *ClientConfigSuite) testApplyWithHeader() {
	cc := ClientConfig{
		Timeout: 15 * time.Second,
		Header: http.Header{
			"Custom": []string{"true"},
		},
	}

	client := new(http.Client)
	suite.Require().NoError(cc.Apply(client))
	suite.addTestRequestAssertions(
		func(candidate *http.Request) {
			suite.Equal("true", candidate.Header.Get("Custom"))
		},
	)

	response := suite.sendRequest(client, "GET", nil)
	suite.Equal(299, response.StatusCode)
}

func (suite *ClientConfigSuite) testApplyCustomRoundTripper() {
	cc := ClientConfig{
		Timeout: 15 * time.Second,
		Header: http.Header{
			"Custom": []string{"true"},
		},
	}

	mockRoundTripper := httpmock.NewRoundTripperSuite(suite)
	client := &http.Client{
		Transport: mockRoundTripper,
	}

	suite.Require().NoError(cc.Apply(client))

	mockRoundTripper.OnMatchAll(httpmock.RequestMatcherFunc(
		func(candidate *http.Request) bool {
			return candidate.Header.Get("Custom") == "true"
		},
	)).Response(&http.Response{
		StatusCode: 299,
	}).Once()

	// this will send things to the mock ...
	response := suite.sendRequest(client, "GET", nil)
	suite.Equal(299, response.StatusCode)
	mockRoundTripper.AssertExpectations()
}

func (suite *ClientConfigSuite) TestApply() {
	suite.Run("NoHeader", suite.testApplyNoHeader)
	suite.Run("WithHeader", suite.testApplyWithHeader)
	suite.Run("CustomRoundTripper", suite.testApplyCustomRoundTripper)
}

func TestClientConfig(t *testing.T) {
	suite.Run(t, new(ClientConfigSuite))
}
