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
	"go.uber.org/fx/fxevent"
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

	// cleanup is the set of closures that ensure any App objects are stopped
	cleanup []func()
}

var _ suite.SetupTestSuite = (*Suite)(nil)
var _ suite.TearDownTestSuite = (*Suite)(nil)

// SetupTest initializes a new viper instance for each test.  If the enclosing
// type needs to implement this method, be sure to invoke this method BEFORE
// any logic that requires the viper environment.
func (suite *Suite) SetupTest() {
	suite.ResetViper()
}

// TearDownTest cleans up any *fxtest.App or *fx.App instances created via
// this type during testing.  If this type is embedded and the enclosing type
// implements TearDownTest, be sure to invoke this method for proper cleanup.
func (suite *Suite) TearDownTest() {
	suite.viper = nil
	for _, f := range suite.cleanup {
		f()
	}

	suite.cleanup = nil
}

// Viper returns the viper instance for the current test.
// Tests that need tighter control over the viper environment may use
// this to bootstrap additional features.
//
// If SetupTest was not called, as would be the case if this type is embedded
// and the enclosing type overrides SetupTest without calling the embedded method,
// the current test will halt.
func (suite *Suite) Viper() *viper.Viper {
	suite.Require().NotNil(
		suite.viper,
		"Viper instance not initialized.  If this type is embedded, you must be sure SetupTest is called.",
	)

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
// If SetupTest was not called, as might be the case if this type was embedded,
// the current test will halt.
//
// The v parameter must be a string, []byte, or an io.Reader.  Any other type
// results in a panic.
func (suite *Suite) YAML(y interface{}) {
	v := suite.Viper()
	v.SetConfigType("yaml")
	suite.Require().NoError(
		v.ReadConfig(suite.reader(y)),
	)
}

// JSON is a shorthand for bootstrapping the current test's viper environment
// with a given JSON configuration.  Invalid JSON will halt the current test.
// If SetupTest was not called, as might be the case if this type was embedded,
// the current test will halt.
//
// The v parameter must be a string, []byte, or an io.Reader.  Any other type
// results in a panic.
func (suite *Suite) JSON(j interface{}) {
	v := suite.Viper()
	v.SetConfigType("json")
	suite.Require().NoError(
		v.ReadConfig(suite.reader(j)),
	)
}

// Option returns an fx.Option that injects the relevant infrastructure for the current test.
// If SetupTest was not called, this method halts the current test.
//
// This method is provided for tests that cannot use the Fx or Fxtest methods.  Those methods
// are the preferred way to create uber/fx App instances for testing.
func (suite *Suite) Option() fx.Option {
	return fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return fxtest.NewTestLogger(suite.T())
		}),
		arrange.ForViper(suite.Viper()),
	)
}

// RequireStart provides the equivalent functionality to fxtest.App.RequireStart, but for a normal
// *fx.App.  This method is useful when an *fx.App is needed for testing instead of an *fxtest.App,
// as is the case with negative testing.
func (suite *Suite) RequireStart(app *fx.App) {
	startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
	defer cancel()
	suite.Require().NoError(
		app.Start(startCtx),
	)
}

// Fxtest is a convenience for doing fxtext.New(...) with the current
// viper environment, test logging, and additional fx.Options.
//
// The returned *fxtest.App will be stopped in TearDownTest.  This ensures
// that any resources held by the App are freed.  Note that you can still
// use RequireStop normally.
func (suite *Suite) Fxtest(more ...fx.Option) *fxtest.App {
	app := fxtest.New(
		suite.T(),
		append(
			[]fx.Option{suite.Option()},
			more...,
		)...,
	)

	suite.cleanup = append(suite.cleanup, app.RequireStop)
	return app
}

// Fx is a convenience for doing fx.New(...) with the current
// viper environment, test logging, and additional fx.Options.
//
// The returned *fx.App will be stopped in TearDownTest.  This ensures
// that any resources held by the App are freed.  Note that you can still
// Stop the App as part of the test.
func (suite *Suite) Fx(more ...fx.Option) *fx.App {
	app := fx.New(
		append(
			[]fx.Option{suite.Option()},
			more...,
		)...,
	)

	suite.cleanup = append(suite.cleanup, func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), app.StopTimeout())
		defer cancel()
		suite.Require().NoError(app.Stop(stopCtx))
	})

	return app
}
