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
