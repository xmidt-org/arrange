package arrangehttp

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

// mockOption is a mock for both server and client options.
type mockOption struct {
	mock.Mock
}

func (m *mockOption) ApplyToServer(s *http.Server) error {
	args := m.Called(s)
	return args.Error(0)
}

func (m *mockOption) ExpectApplyToServer(s *http.Server) *mock.Call {
	return m.On("Apply", s)
}

type mockOptionNoError struct {
	mock.Mock
}

func (m *mockOptionNoError) ApplyToServer(s *http.Server) {
	m.Called(s)
}

func (m *mockOptionNoError) ExpectApplyToServer(s *http.Server) *mock.Call {
	return m.On("Apply", s)
}
