// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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
