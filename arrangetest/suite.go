package arrangetest

import (
	"strings"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// Suite is an embeddable type that makes viper-related tests simpler.
// Embed this type in testify/suite-style test types.
type Suite struct {
	suite.Suite

	// viper is the viper instance for each test
	viper *viper.Viper
}

var _ suite.SetupTestSuite = (*Suite)(nil)

// SetupTest initializes a new viper instance for each test
func (suite *Suite) SetupTest() {
	suite.viper = viper.New()
}

// Viper returns the viper instance for the current test.
// Tests that need tighter control over the viper environment may use
// this to bootstrap additional features.
func (suite *Suite) Viper() *viper.Viper {
	return suite.viper
}

// YAML is a shorthand for bootstrapping the current test's viper environment
// with a given YAML configuration
func (suite *Suite) YAML(v string) {
	suite.viper.SetConfigType("yaml")

	suite.Require().NoError(
		suite.viper.ReadConfig(strings.NewReader(v)),
	)
}

// JSON is a shorthand for bootstrapping the current test's viper environment
// with a given JSON configuration
func (suite *Suite) JSON(v string) {
	suite.viper.SetConfigType("json")

	suite.Require().NoError(
		suite.viper.ReadConfig(strings.NewReader(v)),
	)
}

// Fxtest is a convenience for doing fxtext.New(...) with the current
// viper environment, test logging, and the additional fx.Options
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
// viper environment, test logging, and the additional fx.Options
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
