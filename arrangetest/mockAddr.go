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
