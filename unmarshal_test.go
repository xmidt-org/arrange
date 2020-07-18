package arrange

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// testPrinter is used to redirect fx.App logging to the testing.T object.
// This prevents spamminess when -v is not set.
type testPrinter struct{ *testing.T }

func (tp testPrinter) Printf(msg string, args ...interface{}) {
	tp.T.Logf(msg, args...)
}

type TestConfig struct {
	Name     string
	Age      int
	Interval time.Duration
}

func testUnmarshalValue(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			Unmarshal(TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		actual,
	)
}

func testUnmarshalPointer(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			Unmarshal(&TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		*actual,
	)
}

func testUnmarshalNilPointer(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			Unmarshal((*TestConfig)(nil)),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 0},
		*actual,
	)
}

func TestUnmarshal(t *testing.T) {
	t.Run("Value", testUnmarshalValue)
	t.Run("Pointer", testUnmarshalPointer)
	t.Run("NilPointer", testUnmarshalNilPointer)
}

func testUnmarshalExactValue(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalExact(TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		actual,
	)
}

func testUnmarshalExactValueError(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
nosuch: "foobar"
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		fx.Logger(testPrinter{T: t}),
		fx.Supply(v),
		fx.Provide(
			UnmarshalExact(TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Error(app.Err())
}

func testUnmarshalExactPointer(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalExact(&TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		*actual,
	)
}

func testUnmarshalExactPointerError(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
nosuch: 12345
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		fx.Logger(testPrinter{T: t}),
		fx.Supply(v),
		fx.Provide(
			UnmarshalExact(&TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Error(app.Err())
}

func testUnmarshalExactNilPointer(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalExact((*TestConfig)(nil)),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 0},
		*actual,
	)
}

func testUnmarshalExactNilPointerError(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 16
nosuch: "29384723"
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		fx.Logger(testPrinter{T: t}),
		fx.Supply(v),
		fx.Provide(
			UnmarshalExact((*TestConfig)(nil)),
		),
		fx.Populate(&actual),
	)

	assert.Error(app.Err())
}

func TestUnmarshalExact(t *testing.T) {
	t.Run("Value", testUnmarshalExactValue)
	t.Run("ValueError", testUnmarshalExactValueError)
	t.Run("Pointer", testUnmarshalExactPointer)
	t.Run("PointerError", testUnmarshalExactPointerError)
	t.Run("NilPointer", testUnmarshalExactNilPointer)
	t.Run("NilPointerError", testUnmarshalExactNilPointerError)
}

func testUnmarshalKeyValue(t *testing.T) {
	const yaml = `
testKey:
    name: "testy mctest"
    age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalKey("testKey", TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		actual,
	)
}

func testUnmarshalKeyPointer(t *testing.T) {
	const yaml = `
testKey:
    name: "testy mctest"
    age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalKey("testKey", &TestConfig{Interval: 15 * time.Second}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		*actual,
	)
}

func testUnmarshalKeyNilPointer(t *testing.T) {
	const yaml = `
testKey:
    name: "testy mctest"
    age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalKey("testKey", (*TestConfig)(nil)),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 0},
		*actual,
	)
}

func TestUnmarshalKey(t *testing.T) {
	t.Run("Value", testUnmarshalKeyValue)
	t.Run("Pointer", testUnmarshalKeyPointer)
	t.Run("NilPointer", testUnmarshalKeyNilPointer)
}

func testUnmarshalNamedValue(t *testing.T) {
	const yaml = `
testKey:
    name: "testy mctest"
    age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual TestConfig
	)

	type TestConfigIn struct {
		fx.In
		Actual TestConfig `name:"testKey"`
	}

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalNamed("testKey", TestConfig{Interval: 15 * time.Second}),
		),
		fx.Invoke(func(in TestConfigIn) {
			actual = in.Actual
		}),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		actual,
	)
}

func testUnmarshalNamedPointer(t *testing.T) {
	const yaml = `
testKey:
    name: "testy mctest"
    age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual TestConfig
	)

	type TestConfigIn struct {
		fx.In
		Actual *TestConfig `name:"testKey"`
	}

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalNamed("testKey", &TestConfig{Interval: 15 * time.Second}),
		),
		fx.Invoke(func(in TestConfigIn) {
			actual = *in.Actual
		}),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16, Interval: 15 * time.Second},
		actual,
	)
}

func testUnmarshalNamedNilPointer(t *testing.T) {
	const yaml = `
testKey:
    name: "testy mctest"
    age: 16
`
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual TestConfig
	)

	type TestConfigIn struct {
		fx.In
		Actual *TestConfig `name:"testKey"`
	}

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		fx.Supply(v),
		fx.Provide(
			UnmarshalNamed("testKey", (*TestConfig)(nil)),
		),
		fx.Invoke(func(in TestConfigIn) {
			actual = *in.Actual
		}),
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 16},
		actual,
	)
}

func TestUnmarshalNamed(t *testing.T) {
	t.Run("Value", testUnmarshalNamedValue)
	t.Run("Pointer", testUnmarshalNamedPointer)
	t.Run("NilPointer", testUnmarshalNamedNilPointer)
}
