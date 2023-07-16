package arrangetest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// tb is a helper function that extracts the fxtest.TB from a value.  The value
// may supply a T() *testing.T as a layer of indirection, or it can be a *testing.T
// or *testing.B.
func tb(v any) fxtest.TB {
	type testHolder interface {
		T() *testing.T
	}

	switch vv := v.(type) {
	case testHolder:
		return vv.T()

	case fxtest.TB:
		return vv

	default:
		panic(fmt.Errorf("%T is not a valid test object", v))
	}
}

// NewApp creates an *fxtest.App using the enclosing test.
//
// The t parameter may supply a T() *testing.T method, as in the case of
// a stretchr test suite.  Or, it may implement fxtest.TB directly, as is
// the case with *testing.T and *testing.B.
func NewApp(t any, o ...fx.Option) *fxtest.App {
	return fxtest.New(tb(t), o...)
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

	assert.Error(tb(t), app.Err())
	return app
}
