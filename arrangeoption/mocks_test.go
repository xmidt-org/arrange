package arrangeoption

import (
	"github.com/stretchr/testify/mock"
)

type mockOption[T any] struct {
	mock.Mock
}

func (m *mockOption[T]) Apply(t *T) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *mockOption[T]) ExpectApply(t *T) *mock.Call {
	return m.On("Apply", t)
}

type mockOptionNoError[T any] struct {
	mock.Mock
}

func (m *mockOptionNoError[T]) Apply(t *T) {
	m.Called(t)
}

func (m *mockOptionNoError[T]) ExpectApply(t *T) *mock.Call {
	return m.On("Apply", t)
}
