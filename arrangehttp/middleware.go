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
