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
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetest"
	"github.com/xmidt-org/arrange/arrangetls"
)

type ListenerSuite struct {
	arrangetls.Suite
}

func (suite *ListenerSuite) testDefaultListenerFactoryBasic() {
	var (
		factory DefaultListenerFactory
		server  = &http.Server{
			Addr: ":0",
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	suite.Require().NoError(err)
	suite.Require().NotNil(listener)
	suite.NotNil(listener.Addr())
	listener.Close()
}

func (suite *ListenerSuite) testDefaultListenerFactoryWithTLS() {
	var (
		factory DefaultListenerFactory
		server  = &http.Server{
			Addr:      ":0",
			TLSConfig: suite.TLSConfig(),
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	suite.Require().NoError(err)
	suite.Require().NotNil(listener)
	suite.NotNil(listener.Addr())
	listener.Close()
}

func (suite *ListenerSuite) testDefaultListenerFactoryError() {
	var (
		factory = DefaultListenerFactory{
			Network: "this is a bad network",
		}

		server = &http.Server{
			Addr: ":0",
		}
	)

	listener, err := factory.Listen(context.Background(), server)
	suite.Error(err)

	if !suite.Nil(listener) {
		// cleanup if the assertion fails, meaning the factory incorrectly
		// returned a non-nil listener AND a non-nil error.
		listener.Close()
	}
}

func (suite *ListenerSuite) TestDefaultListenerFactory() {
	suite.Run("Basic", suite.testDefaultListenerFactoryBasic)
	suite.Run("WithTLS", suite.testDefaultListenerFactoryWithTLS)
	suite.Run("Error", suite.testDefaultListenerFactoryError)
}

func (suite *ListenerSuite) testNewListenerNilListenerFactory() {
	var (
		capture = make(chan net.Addr, 1)
		l, err  = NewListener(
			context.Background(),
			nil,
			&http.Server{
				Addr: ":0",
			},
			arrangetest.ListenCapture(capture),
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()
	actual := arrangetest.ListenReceive(suite, capture, time.Second)
	suite.Equal(l.Addr(), actual)
}

func (suite *ListenerSuite) testNewListenerCustomListenerFactory() {
	var (
		capture = make(chan net.Addr, 1)
		l, err  = NewListener(
			context.Background(),
			ServerConfig{},
			&http.Server{
				Addr: ":0",
			},
			arrangetest.ListenCapture(capture),
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()
	actual := arrangetest.ListenReceive(suite, capture, time.Second)
	suite.Equal(l.Addr(), actual)
}

func (suite *ListenerSuite) TestNewListener() {
	suite.Run("NilListenerFactory", suite.testNewListenerNilListenerFactory)
	suite.Run("CustomListenerFactory", suite.testNewListenerCustomListenerFactory)
}

func TestListener(t *testing.T) {
	suite.Run(t, new(ListenerSuite))
}
