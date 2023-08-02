// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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
