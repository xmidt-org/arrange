package arrangetest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
)

// SuiteTestSuite embeds Suite in the expected way and verifies
// that the suite lifecycle works properly
type SuiteTestSuite struct {
	Suite
}

func (suite *SuiteTestSuite) TestYAML() {
	suite.Require().NotNil(suite.Viper())
	suite.YAML(`
keys:
  - value1
  - value2
  - value3
`,
	)

	suite.Equal(
		[]string{"value1", "value2", "value3"},
		suite.Viper().GetStringSlice("keys"),
	)
}

func (suite *SuiteTestSuite) TestJSON() {
	suite.Require().NotNil(suite.Viper())
	suite.JSON(`{
"keys": ["value1", "value2", "value3"]
	}`)

	suite.Equal(
		[]string{"value1", "value2", "value3"},
		suite.Viper().GetStringSlice("keys"),
	)
}

func (suite *SuiteTestSuite) TestFxtest() {
	var component int

	app := suite.Fxtest(
		fx.Provide(
			func() int {
				return 123
			},
		),
		fx.Populate(&component),
	)

	suite.Equal(123, component)
	app.RequireStart()
	app.RequireStop()
}

func (suite *SuiteTestSuite) TestFx() {
	var component int

	app := suite.Fx(
		fx.Provide(
			func() int {
				return 123
			},
		),
		fx.Populate(&component),
	)

	suite.Equal(123, component)
	suite.NoError(app.Start(context.Background()))
	suite.NoError(app.Stop(context.Background()))
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteTestSuite))
}
