package arrange

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func testViperUnmarshalerNoOptions(t *testing.T) {
	type Data struct {
		Value int
	}

	const yaml = `
value: 123
test:
  value: 456
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		output  bytes.Buffer
		v       = viper.New()

		data Data
		vu   = ViperUnmarshaler{
			Viper:   v,
			Printer: NewPrinterWriter(&output),
		}
	)

	v.SetConfigType("yaml")
	require.NoError(
		v.ReadConfig(strings.NewReader(yaml)),
	)

	output.Reset()
	require.NoError(vu.Unmarshal(&data))
	assert.Equal(Data{Value: 123}, data)
	assert.NotEmpty(output.String())

	output.Reset()
	data = Data{}
	require.NoError(vu.UnmarshalKey("test", &data))
	assert.Equal(Data{Value: 456}, data)
	assert.NotEmpty(output.String())
}

func testViperUnmarshalerWithOptions(t *testing.T) {
	type Data struct {
		Value int
	}

	const yaml = `
value: 123
test:
  value: 456
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		output  bytes.Buffer
		v       = viper.New()

		optionCalled bool
		option       = viper.DecoderConfigOption(func(*mapstructure.DecoderConfig) {
			optionCalled = true
		})

		data Data
		vu   = ViperUnmarshaler{
			Viper:   v,
			Options: []viper.DecoderConfigOption{option},
			Printer: NewPrinterWriter(&output),
		}
	)

	v.SetConfigType("yaml")
	require.NoError(
		v.ReadConfig(strings.NewReader(yaml)),
	)

	output.Reset()
	require.NoError(vu.Unmarshal(&data))
	assert.Equal(Data{Value: 123}, data)
	assert.NotEmpty(output.String())
	assert.True(optionCalled)

	output.Reset()
	optionCalled = false
	data = Data{}
	require.NoError(vu.UnmarshalKey("test", &data))
	assert.Equal(Data{Value: 456}, data)
	assert.NotEmpty(output.String())
	assert.True(optionCalled)
}

func TestViperUnmarshaler(t *testing.T) {
	t.Run("NoOptions", testViperUnmarshalerNoOptions)
	t.Run("WithOptions", testViperUnmarshalerWithOptions)
}

func testForViperNil(t *testing.T) {
	var (
		assert = assert.New(t)

		unmarshaler Unmarshaler
	)

	app := fx.New(
		TestLogger(t),
		ForViper(nil),
		fx.Populate(&unmarshaler),
	)

	assert.Error(app.Err())
}

func testForViperNoOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v           = viper.New()
		unmarshaler Unmarshaler
	)

	fxtest.New(
		t,
		TestLogger(t),
		ForViper(v),
		fx.Populate(&unmarshaler),
	)

	vu, ok := unmarshaler.(ViperUnmarshaler)
	require.True(ok)
	assert.True(v == vu.Viper)
	assert.Empty(vu.Options)
	assert.NotNil(vu.Printer)
}

func testForViperWithOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		option1Called bool
		option1       = func(*mapstructure.DecoderConfig) {
			option1Called = true
		}

		option2Called bool
		option2       = func(*mapstructure.DecoderConfig) {
			option2Called = true
		}

		v           = viper.New()
		unmarshaler Unmarshaler
	)

	fxtest.New(
		t,
		TestLogger(t),
		ForViper(v, option1),
		fx.Provide(
			func() []viper.DecoderConfigOption {
				return []viper.DecoderConfigOption{option2}
			},
		),
		fx.Populate(&unmarshaler),
	)

	vu, ok := unmarshaler.(ViperUnmarshaler)
	require.True(ok)
	assert.True(v == vu.Viper)
	assert.Len(vu.Options, 2)
	assert.NotNil(vu.Printer)

	vu.Unmarshal(nil)
	assert.True(option1Called)
	assert.True(option2Called)
}

func TestForViper(t *testing.T) {
	t.Run("Nil", testForViperNil)
	t.Run("NoOptions", testForViperNoOptions)
	t.Run("WithOptions", testForViperWithOptions)
}
