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

package arrangehttp

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
