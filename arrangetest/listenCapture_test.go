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
	actual, ok := ListenReceive(ch, time.Second)
	suite.True(ok)
	suite.Same(expected, actual)
}

func (suite *ListenSuite) testListenReceiveTimeout() {
	var (
		ch = make(chan net.Addr, 1)
		t  = make(chan time.Time, 1)
	)

	t <- time.Now() // won't block, buffer size is 1
	actual, ok := listenReceive(ch, t)
	suite.False(ok)
	suite.Nil(actual)
}

func (suite *ListenSuite) TestListenReceive() {
	suite.Run("Success", suite.testListenReceiveSuccess)
	suite.Run("Timeout", suite.testListenReceiveTimeout)
}

func TestListen(t *testing.T) {
	suite.Run(t, new(ListenSuite))
}
