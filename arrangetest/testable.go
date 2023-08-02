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

import (
	"fmt"
	"testing"
)

// Testable is the minimal interface required for assertions and testing.
// This interface is implemented by several libraries.
type Testable interface {
	Logf(string, ...interface{})
	Errorf(string, ...interface{})
	FailNow()
}

// AsTestable converts a value into a Testable.  The v parameter
// may be a *testing.T, *testing.B, or a type that provides a T() *testing.T method.
//
// If v cannot be coerced into a Testable, this function panics.
func AsTestable(v any) Testable {
	if tt, ok := v.(Testable); ok {
		return tt
	}

	type testHolder interface {
		T() *testing.T
	}

	if th, ok := v.(testHolder); ok {
		return th.T()
	}

	panic(fmt.Errorf("%T cannot be converted into a Testable", v))
}
