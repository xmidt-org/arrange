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
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MockListenerSuite struct {
	suite.Suite
}

func (suite *MockListenerSuite) TestExpectAccept() {
	var (
		expected    = new(net.IPConn)
		expectedErr = errors.New("expected")
		m           = new(MockListener)
	)

	m.ExpectAccept(expected, expectedErr).Once()
	actual, actualErr := m.Accept()
	suite.Same(expected, actual)
	suite.Same(expectedErr, actualErr)
	m.AssertExpectations(suite.T())
}

func (suite *MockListenerSuite) TestExpectClose() {
	var (
		expectedErr = errors.New("expected")
		m           = new(MockListener)
	)

	m.ExpectClose(expectedErr).Once()
	m.ExpectClose(nil).Once()
	suite.Same(expectedErr, m.Close())
	suite.NoError(m.Close())

	m.AssertExpectations(suite.T())
}

func (suite *MockListenerSuite) TestExpectAddr() {
	var (
		expected = new(net.IPAddr)
		m        = new(MockListener)
	)

	m.ExpectAddr(expected).Once()
	suite.Same(expected, m.Addr())

	m.AssertExpectations(suite.T())
}

func TestMockListener(t *testing.T) {
	suite.Run(t, new(MockListenerSuite))
}
