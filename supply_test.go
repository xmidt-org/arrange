package arrange

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func testSupplyViperOnly(t *testing.T) {
	var (
		assert = assert.New(t)

		expected = viper.New()
		actual   *viper.Viper
	)

	fxtest.New(
		t,
		Supply(expected),
		fx.Populate(&actual),
	)

	assert.Equal(expected, actual)
}

func testSupplyWithOptions(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedViper   = viper.New()
		expectedOptions = []viper.DecoderConfigOption{
			Exact,
			WeaklyTypedInput(false),
		}

		actualViper   *viper.Viper
		actualOptions []viper.DecoderConfigOption
	)

	fxtest.New(
		t,
		Supply(expectedViper, expectedOptions...),
		fx.Populate(&actualViper, &actualOptions),
	)

	assert.Equal(expectedViper, actualViper)
	assert.Equal(expectedOptions, actualOptions)
}

func testSupplyNilViper(t *testing.T) {
	var (
		assert = assert.New(t)

		actual *viper.Viper
	)

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		TestLogger(t),
		Supply(nil),
		fx.Populate(&actual),
	)

	assert.Error(app.Err())
}

func TestSupply(t *testing.T) {
	t.Run("ViperOnly", testSupplyViperOnly)
	t.Run("WithOptions", testSupplyWithOptions)
	t.Run("NilViper", testSupplyNilViper)
}
