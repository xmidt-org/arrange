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
