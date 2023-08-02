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

package arrange

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestConditional(t *testing.T) {
	var (
		assert = assert.New(t)

		ifTrue     bool
		ifNotFalse bool
	)

	fxtest.New(
		t,
		If(true).Then(
			fx.Invoke(func() {
				ifTrue = true
			}),
		),
		If(false).Then(
			fx.Invoke(func() error {
				return errors.New("If(false) should not return any options")
			}),
		),
		IfNot(true).Then(
			fx.Invoke(func() error {
				return errors.New("IfNot(true) should not return any options")
			}),
		),
		IfNot(false).Then(
			fx.Invoke(func() {
				ifNotFalse = true
			}),
		),
	)

	assert.True(ifTrue)
	assert.True(ifNotFalse)
}
