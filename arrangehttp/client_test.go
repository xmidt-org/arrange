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
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/httpaux/httpmock"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type ClientSuite struct {
	suite.Suite
}

func (suite *ClientSuite) TestNewClient() {
	mockTransport := httpmock.NewRoundTripperSuite(suite)
	cc := ClientConfig{
		Timeout: 27 * time.Minute,
		Header: http.Header{
			"Custom": []string{"true"},
		},
	}

	client, err := NewClient(cc,
		AsOption[http.Client](func(c *http.Client) {
			c.Transport = mockTransport
		}),
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(client)

	mockTransport.OnMatchAll(httpmock.RequestMatcherFunc(func(candidate *http.Request) bool {
		return "true" == candidate.Header.Get("Custom")
	})).Response(&http.Response{
		StatusCode: 299,
	}).Once()

	response, err := client.Do(&http.Request{
		Method: "GET",
		URL: &url.URL{
			Path: "/",
		},
	})

	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Equal(299, response.StatusCode)

	mockTransport.AssertExpectations()
}

func (suite *ClientSuite) testProvideClientNoName() {
	app := fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: zap.NewNop()}
		}),
		ProvideClient(""), // should result in an error
	)

	suite.Error(app.Err())
}

func (suite *ClientSuite) testProvideClientSimple() {
	var client *http.Client
	app := fxtest.New(
		suite.T(),
		ProvideClient("client"),
		fx.Populate(
			fx.Annotate(
				&client,
				arrange.Tags().Name("client").ParamTags(),
			),
		),
	)

	app.RequireStart()
	suite.Require().NotNil(client)
	app.RequireStop()
}

func (suite *ClientSuite) testProvideClientWithConfig() {
	var client *http.Client
	mockTransport := httpmock.NewRoundTripperSuite(suite)
	app := fxtest.New(
		suite.T(),
		fx.Supply(
			fx.Annotated{
				Name: "client.config",
				Target: ClientConfig{
					Timeout: 15 * time.Second,
				},
			},
		),
		fx.Provide(
			fx.Annotate(
				func() Option[http.Client] {
					return AsOption[http.Client](func(c *http.Client) {
						c.Transport = mockTransport
					})
				},
				arrange.Tags().Group("client.options").ResultTags(),
			),
		),
		ProvideClient("client"),
		fx.Populate(
			fx.Annotate(
				&client,
				arrange.Tags().Name("client").ParamTags(),
			),
		),
	)

	app.RequireStart()
	suite.Require().NotNil(client)
	app.RequireStop()

	suite.Equal(15*time.Second, client.Timeout)
	suite.Same(mockTransport, client.Transport)
	mockTransport.AssertExpectations()
}

func (suite *ClientSuite) TestProvideClient() {
	suite.Run("NoName", suite.testProvideClientNoName)
	suite.Run("Simple", suite.testProvideClientSimple)
	suite.Run("WithConfig", suite.testProvideClientWithConfig)
}

func (suite *ClientSuite) NewClient() (*http.Client, error) {
	return &http.Client{
		Timeout: 167 * time.Second,
	}, nil
}

func (suite *ClientSuite) TestProvideClientCustom() {
	var client *http.Client
	mockTransport := httpmock.NewRoundTripperSuite(suite)
	app := fxtest.New(
		suite.T(),
		fx.Supply(
			fx.Annotated{
				Name:   "client.config",
				Target: suite,
			},
		),
		fx.Provide(
			fx.Annotate(
				func() Option[http.Client] {
					return AsOption[http.Client](func(c *http.Client) {
						c.Transport = mockTransport
					})
				},
				arrange.Tags().Group("client.options").ResultTags(),
			),
		),
		ProvideClientCustom[*ClientSuite]("client"),
		fx.Populate(
			fx.Annotate(
				&client,
				arrange.Tags().Name("client").ParamTags(),
			),
		),
	)

	app.RequireStart()
	suite.Require().NotNil(client)
	app.RequireStop()

	suite.Equal(167*time.Second, client.Timeout)
	suite.Same(mockTransport, client.Transport)
	mockTransport.AssertExpectations()
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}
