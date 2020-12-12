package arrangetest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// Suite is an embeddable type that makes viper-related tests simpler.
// Embed this type in testify/suite-style test types.
//
// See: https://pkg.go.dev/github.com/stretchr/testify/suite#Suite
type Suite struct {
	suite.Suite

	// viper is the viper instance for each test
	viper *viper.Viper
}

var _ suite.SetupTestSuite = (*Suite)(nil)

// SetupTest initializes a new viper instance for each test.  If the enclosing
// type needs to implement this method, be sure to invoke this method BEFORE
// any logic that requires the viper environment.
func (suite *Suite) SetupTest() {
	suite.ResetViper()
}

// Viper returns the viper instance for the current test.
// Tests that need tighter control over the viper environment may use
// this to bootstrap additional features.
func (suite *Suite) Viper() *viper.Viper {
	return suite.viper
}

// ResetViper associates a brand new viper instance with this test.
// This method is useful when running subtests, since SetupTest doesn't
// run for each subtest.
func (suite *Suite) ResetViper() *viper.Viper {
	suite.viper = viper.New()
	return suite.viper
}

func (suite *Suite) reader(v interface{}) io.Reader {
	switch src := v.(type) {
	case []byte:
		return bytes.NewReader(src)

	case string:
		return strings.NewReader(src)

	case io.Reader:
		return src

	default:
		panic(fmt.Errorf("%T is not support as a source of configuration", v))
	}
}

// YAML is a shorthand for bootstrapping the current test's viper environment
// with a given YAML configuration.  Invalid YAML will halt the current test.
//
// The v parameter must be a string, []byte, or an io.Reader.  Any other type
// results in a panic.
func (suite *Suite) YAML(v interface{}) {
	suite.viper.SetConfigType("yaml")
	suite.Require().NoError(
		suite.viper.ReadConfig(suite.reader(v)),
	)
}

// JSON is a shorthand for bootstrapping the current test's viper environment
// with a given JSON configuration.  Invalid JSON will halt the current test.
//
// The v parameter must be a string, []byte, or an io.Reader.  Any other type
// results in a panic.
func (suite *Suite) JSON(v interface{}) {
	suite.viper.SetConfigType("json")
	suite.Require().NoError(
		suite.viper.ReadConfig(suite.reader(v)),
	)
}

// RequireStart provides the same functionality as fxtest.App.RequireStart, but for
// either an *fx.App or an *fxtest.App.
//
// If v is not an *fxtest.App or an *fx.App, this method panics.
func (suite *Suite) RequireStart(v interface{}) {
	switch app := v.(type) {
	case *fxtest.App:
		app.RequireStart()

	case *fx.App:
		startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
		defer cancel()
		suite.Require().NoError(app.Start(startCtx))

	default:
		panic(fmt.Errorf("%T is not an *fxtest.App or an *fx.App", v))
	}
}

// RequireStop provides the same functionality as fxtest.App.RequireStop, but for
// either an *fx.App or an *fxtest.App.  This method ensures that a test failure
// is recorded if an app doesn't stop properly.  When used in a defer, this method
// not only ensures the app is stopped but also marks the current test as failed
// if the app does not stop cleanly.
//
// If v is not an *fxtest.App or an *fx.App, this method panics.
func (suite *Suite) RequireStop(v interface{}) {
	switch app := v.(type) {
	case *fxtest.App:
		app.RequireStop()

	case *fx.App:
		stopCtx, cancel := context.WithTimeout(context.Background(), app.StopTimeout())
		defer cancel()
		suite.Require().NoError(app.Stop(stopCtx))

	default:
		panic(fmt.Errorf("%T is not an *fxtest.App or an *fx.App", v))
	}
}

// EnsureStop is like RequireStop, except that it does not mark the test as failed
// if the app does not stop cleanly.  Instead, this method logs any error from Stop.
//
// In general, RequireStop is preferred over this method.  However, there are cases
// when testing failure conditions where errors are expected during the test but the
// app needs to be stopped so that it doesn't continue consuming resources after
// the test.  Placing this method in a defer call accomplishes that.
//
// As with RequireStop, this method panics if v is not an *fx.App or *fxtest.App.
func (suite *Suite) EnsureStop(v interface{}) {
	var (
		stop    func(context.Context) error
		stopCtx context.Context
		cancel  func()
	)

	switch app := v.(type) {
	case *fxtest.App:
		stopCtx, cancel = context.WithTimeout(context.Background(), app.StopTimeout())
		stop = app.Stop

	case *fx.App:
		stopCtx, cancel = context.WithTimeout(context.Background(), app.StopTimeout())
		stop = app.Stop

	default:
		panic(fmt.Errorf("%T is not an *fxtest.App or an *fx.App", v))
	}

	defer cancel()
	err := stop(stopCtx)
	if err != nil {
		suite.T().Logf("%T failed to stop: %s", v, err)
	}
}

// Fxtest is a convenience for doing fxtext.New(...) with the current
// viper environment, test logging, and additional fx.Options
func (suite *Suite) Fxtest(more ...fx.Option) *fxtest.App {
	return fxtest.New(
		suite.T(),
		append(
			[]fx.Option{
				arrange.TestLogger(suite.T()),
				arrange.ForViper(suite.viper),
			},
			more...,
		)...,
	)
}

// Fx is a convenience for doing fx.New(...) with the current
// viper environment, test logging, and additional fx.Options
func (suite *Suite) Fx(more ...fx.Option) *fx.App {
	return fx.New(
		append(
			[]fx.Option{
				arrange.TestLogger(suite.T()),
				arrange.ForViper(suite.viper),
			},
			more...,
		)...,
	)
}
