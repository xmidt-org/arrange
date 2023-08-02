// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangetest

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// NewApp creates an *fxtest.App using the enclosing test.
//
// The t parameter may supply a T() *testing.T method, as in the case of
// a stretchr test suite.  Or, it may implement fxtest.TB directly, as is
// the case with *testing.T and *testing.B.
func NewApp(t any, o ...fx.Option) *fxtest.App {
	return fxtest.New(AsTestable(t), o...)
}

// NewErrApp creates an *fx.App which is expected to fail during construction.
// Prior to returning, this function asserts that there was an error.  The *fx.App
// is returned for any further assertions.  The t parameter has the same restrictions
// as NewApp.
//
// Since an error is assumed to happen, the returned app has logging silenced.
func NewErrApp(t any, o ...fx.Option) *fx.App {
	app := fx.New(
		append(
			o,
			fx.NopLogger,
		)...,
	)

	assert.Error(AsTestable(t), app.Err())
	return app
}
