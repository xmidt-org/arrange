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

import "github.com/stretchr/testify/mock"

// MockAddr is a mocked net.Addr.
type MockAddr struct {
	mock.Mock
}

func (m *MockAddr) Network() string {
	return m.Called().String(0)
}

func (m *MockAddr) ExpectNetwork(n string) *mock.Call {
	return m.On("Network").Return(n)
}

func (m *MockAddr) String() string {
	return m.Called().String(0)
}

func (m *MockAddr) ExpectString(n string) *mock.Call {
	return m.On("String").Return(n)
}
