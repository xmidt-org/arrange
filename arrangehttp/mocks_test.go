package arrangehttp

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

type mockServerOption struct {
	mock.Mock
}

func (m *mockServerOption) Apply(s *http.Server) error {
	args := m.Called(s)
	return args.Error(0)
}

func (m *mockServerOption) ExpectApply(s *http.Server) *mock.Call {
	return m.On("Apply", s)
}

type mockServerOptionNoError struct {
	mock.Mock
}

func (m *mockServerOptionNoError) Apply(s *http.Server) {
	m.Called(s)
}

func (m *mockServerOptionNoError) ExpectApply(s *http.Server) *mock.Call {
	return m.On("Apply", s)
}
