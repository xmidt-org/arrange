package arrangetest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
)

// SuiteTestSuite embeds Suite in the expected way and verifies
// that the suite lifecycle works properly
type SuiteTestSuite struct {
	Suite
}

func (suite *SuiteTestSuite) TestResetViper() {
	original := suite.Viper()
	suite.Require().NotNil(original, "the test setup did not run")

	reset := suite.ResetViper()
	suite.True(original != reset)
	suite.True(suite.Viper() == reset)
}

func (suite *SuiteTestSuite) TestYAML() {
	suite.Run("string", func() {
		suite.ResetViper()
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
	})

	suite.Run("[]byte", func() {
		suite.ResetViper()
		suite.YAML([]byte(`
keys:
  - value1
  - value2
  - value3
`),
		)

		suite.Equal(
			[]string{"value1", "value2", "value3"},
			suite.Viper().GetStringSlice("keys"),
		)
	})

	suite.Run("io.Reader", func() {
		suite.ResetViper()
		suite.YAML(strings.NewReader(`
keys:
  - value1
  - value2
  - value3
`),
		)

		suite.Equal(
			[]string{"value1", "value2", "value3"},
			suite.Viper().GetStringSlice("keys"),
		)
	})

	suite.Run("InvalidType", func() {
		suite.ResetViper()
		suite.Panics(func() {
			suite.YAML(123)
		})
	})
}

func (suite *SuiteTestSuite) TestJSON() {
	suite.Run("string", func() {
		suite.ResetViper()
		suite.JSON(`{
"keys": ["value1", "value2", "value3"]
	}`)

		suite.Equal(
			[]string{"value1", "value2", "value3"},
			suite.Viper().GetStringSlice("keys"),
		)
	})

	suite.Run("[]byte", func() {
		suite.ResetViper()
		suite.JSON([]byte(`{
"keys": ["value1", "value2", "value3"]
	}`))

		suite.Equal(
			[]string{"value1", "value2", "value3"},
			suite.Viper().GetStringSlice("keys"),
		)
	})

	suite.Run("io.Reader", func() {
		suite.ResetViper()
		suite.JSON(strings.NewReader(`{
"keys": ["value1", "value2", "value3"]
	}`))

		suite.Equal(
			[]string{"value1", "value2", "value3"},
			suite.Viper().GetStringSlice("keys"),
		)
	})

	suite.Run("InvalidType", func() {
		suite.ResetViper()
		suite.Panics(func() {
			suite.JSON(123)
		})
	})
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
	suite.RequireStart(app)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteTestSuite))
}
