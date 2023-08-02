// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangetest

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MockAddrSuite struct {
	suite.Suite
}

func (suite *MockAddrSuite) TestExpectNetwork() {
	m := new(MockAddr)
	m.ExpectNetwork("foo").Once()
	suite.Equal("foo", m.Network())
	m.AssertExpectations(suite.T())
}

func (suite *MockAddrSuite) TestExpectString() {
	m := new(MockAddr)
	m.ExpectString("foo").Once()
	suite.Equal("foo", m.String())
	m.AssertExpectations(suite.T())
}

func TestMockAddr(t *testing.T) {
	suite.Run(t, new(MockAddrSuite))
}
