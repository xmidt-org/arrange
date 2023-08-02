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

package arrangetest

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ListenSuite struct {
	suite.Suite
}

func (suite *ListenSuite) TestListenCapture() {
	var (
		expected = new(MockAddr)

		l  = new(MockListener)
		ch = make(chan net.Addr, 1)
		m  = ListenCapture(ch)
	)

	l.ExpectAddr(expected).Once()
	decorated := m(l)
	suite.Same(l, decorated)

	select {
	case actual := <-ch:
		suite.Same(expected, actual)

	case <-time.After(time.Second):
		suite.Fail("Did not receive the listen address")

	}

	l.AssertExpectations(suite.T())
}

func (suite *ListenSuite) testListenReceiveSuccess() {
	var (
		expected = new(MockAddr)
		ch       = make(chan net.Addr, 1)
	)

	ch <- expected // won't block, buffer size is 1
	actual := ListenReceive(suite, ch, time.Second)
	suite.Same(expected, actual)
}

func (suite *ListenSuite) testListenReceiveFail() {
	var (
		mockT = new(mockTestable)
		ch    = make(chan net.Addr, 1)
	)

	mockT.ExpectAnyErrorf()
	mockT.ExpectFailNow()

	actual := ListenReceive(mockT, ch, time.Millisecond)
	suite.Nil(actual)
	mockT.AssertExpectations(suite.T())
}

func (suite *ListenSuite) TestListenReceive() {
	suite.Run("Success", suite.testListenReceiveSuccess)
	suite.Run("Fail", suite.testListenReceiveFail)
}

func TestListen(t *testing.T) {
	suite.Run(t, new(ListenSuite))
}
