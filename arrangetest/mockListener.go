// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangetest

import (
	"net"

	"github.com/stretchr/testify/mock"
)

// MockListener is a mocked net.Listener.
type MockListener struct {
	mock.Mock
}

func (m *MockListener) Accept() (net.Conn, error) {
	args := m.Called()
	c, _ := args.Get(0).(net.Conn)
	return c, args.Error(1)
}

func (m *MockListener) ExpectAccept(c net.Conn, err error) *mock.Call {
	return m.On("Accept").Return(c, err)
}

func (m *MockListener) Close() error {
	return m.Called().Error(0)
}

func (m *MockListener) ExpectClose(err error) *mock.Call {
	return m.On("Close").Return(err)
}

func (m *MockListener) Addr() net.Addr {
	args := m.Called()
	a, _ := args.Get(0).(net.Addr)
	return a
}

func (m *MockListener) ExpectAddr(a net.Addr) *mock.Call {
	return m.On("Addr").Return(a)
}
