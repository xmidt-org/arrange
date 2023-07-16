package arrangetest

import "github.com/stretchr/testify/mock"

type mockTB struct {
	mock.Mock
}

func (m *mockTB) Logf(format string, args ...any) {
	m.Called(format, args)
}

func (m *mockTB) ExpectAnyLogf() *mock.Call {
	return m.On(
		"Logf",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func([]any) bool { return true }),
	)
}

func (m *mockTB) Errorf(format string, args ...any) {
	m.Called(format, args)
}

func (m *mockTB) ExpectAnyErrorf() *mock.Call {
	return m.On(
		"Errorf",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func([]any) bool { return true }),
	)
}

func (m *mockTB) FailNow() {
	m.Called()
}

func (m *mockTB) ExpectFailNow() *mock.Call {
	return m.On("FailNow")
}
