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

type mockTestable struct {
	mock.Mock
}

func (m *mockTestable) Logf(format string, args ...any) {
	m.Called(format, args)
}

func (m *mockTestable) ExpectAnyLogf() *mock.Call {
	return m.On(
		"Logf",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func([]any) bool { return true }),
	)
}

func (m *mockTestable) Errorf(format string, args ...any) {
	m.Called(format, args)
}

func (m *mockTestable) ExpectAnyErrorf() *mock.Call {
	return m.On(
		"Errorf",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func([]any) bool { return true }),
	)
}

func (m *mockTestable) FailNow() {
	m.Called()
}

func (m *mockTestable) ExpectFailNow() *mock.Call {
	return m.On("FailNow")
}
