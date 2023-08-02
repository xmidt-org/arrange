// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangehttp

// Middleware is the underlying type for decorators.
type Middleware[T any] interface {
	~func(T) T
}

// ApplyMiddleware handles decorating a target type T.  Middleware
// executes in the order declared to this function.
func ApplyMiddleware[T any, M Middleware[T]](t T, m ...M) T {
	for i := len(m) - 1; i >= 0; i-- {
		t = m[i](t)
	}

	return t
}
