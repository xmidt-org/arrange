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
