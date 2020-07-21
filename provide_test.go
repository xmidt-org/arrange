package arrange

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func testProvideSuccess(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 64
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		value           TestConfig
		initiallyNotNil *AnotherConfig
		initiallyNil    *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		Supply(v),
		Provide(TestConfig{Interval: 15 * time.Second}),
		Provide(&AnotherConfig{Interval: 15 * time.Second}),
		Provide((*TestConfig)(nil)),
		fx.Populate(&value, &initiallyNil, &initiallyNotNil),
	)

	require.NotNil(initiallyNotNil)
	require.NotNil(initiallyNil)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 64, Interval: 15 * time.Second},
		value,
	)

	assert.Equal(
		AnotherConfig{Name: "testy mctest", Age: 64, Interval: 15 * time.Second},
		*initiallyNotNil,
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 64},
		*initiallyNil,
	)
}

func testProvideExact(t *testing.T) {
	const yaml = `
name: "testy mctest"
age: 64
nosuch: asdfasdfasdf
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		value TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		fx.Logger(testPrinter{T: t}),
		Supply(v, Exact),
		Provide(TestConfig{}),
		fx.Populate(&value),
	)

	assert.Error(app.Err())
}

func TestProvide(t *testing.T) {
	t.Run("Success", testProvideSuccess)
	t.Run("Exact", testProvideExact)
}

func testProvideKeySuccess(t *testing.T) {
	const yaml = `
test:
  name: "testy mctest"
  age: 64
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		value           TestConfig
		initiallyNotNil *AnotherConfig
		initiallyNil    *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		Supply(v),
		ProvideKey("test", TestConfig{Interval: 15 * time.Second}),

		// need to give this another type, since it will conflict with another *TestConfig component
		ProvideKey("test", &AnotherConfig{Interval: 15 * time.Second}),

		ProvideKey("test", (*TestConfig)(nil)),
		fx.Populate(&value, &initiallyNil, &initiallyNotNil),
	)

	require.NotNil(initiallyNotNil)
	require.NotNil(initiallyNil)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 64, Interval: 15 * time.Second},
		value,
	)

	assert.Equal(
		AnotherConfig{Name: "testy mctest", Age: 64, Interval: 15 * time.Second},
		*initiallyNotNil,
	)

	assert.Equal(
		TestConfig{Name: "testy mctest", Age: 64},
		*initiallyNil,
	)
}

func testProvideKeyExact(t *testing.T) {
	const yaml = `
test:
  name: "testy mctest"
  age: 64
  nosuch: asdfasdfasdf
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		value TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		fx.Logger(testPrinter{T: t}),
		Supply(v, Exact),
		ProvideKey("test", TestConfig{}),
		fx.Populate(&value),
	)

	assert.Error(app.Err())
}

func TestProvideKey(t *testing.T) {
	t.Run("Success", testProvideKeySuccess)
	t.Run("Exact", testProvideKeyExact)
}

func testProvideNamedSuccess(t *testing.T) {
	const yaml = `
value:
  name: "test #1"
  age: 17
initiallyNotNil:
  name: "test #2"
  age: 34
initiallyNil:
  name: "test #3"
  age: 58
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		value           TestConfig
		initiallyNotNil *TestConfig
		initiallyNil    *TestConfig
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	type In struct {
		fx.In

		Value           TestConfig  `name:"value"`
		InitiallyNotNil *TestConfig `name:"initiallyNotNil"`
		InitiallyNil    *TestConfig `name:"initiallyNil"`
	}

	fxtest.New(
		t,
		Supply(v),
		ProvideNamed("value", TestConfig{Interval: 56 * time.Minute}),
		ProvideNamed("initiallyNotNil", &TestConfig{Interval: 17 * time.Millisecond}),
		ProvideNamed("initiallyNil", (*TestConfig)(nil)),
		fx.Invoke(
			func(in In) {
				value = in.Value
				initiallyNotNil = in.InitiallyNotNil
				initiallyNil = in.InitiallyNil
			},
		),
	)

	require.NotNil(initiallyNotNil)
	require.NotNil(initiallyNil)

	assert.Equal(
		TestConfig{Name: "test #1", Age: 17, Interval: 56 * time.Minute},
		value,
	)

	assert.Equal(
		TestConfig{Name: "test #2", Age: 34, Interval: 17 * time.Millisecond},
		*initiallyNotNil,
	)

	assert.Equal(
		TestConfig{Name: "test #3", Age: 58},
		*initiallyNil,
	)
}

func testProvideNamedExact(t *testing.T) {
	const yaml = `
test:
  name: "testy mctest"
  age: 64
  nosuch: asdfasdfasdf
`

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	type In struct {
		fx.In

		Value TestConfig `name:"test"`
	}

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		fx.Logger(testPrinter{T: t}),
		Supply(v, Exact),
		ProvideNamed("test", TestConfig{}),
		fx.Invoke(
			func(in In) error {
				return errors.New("the invoke should not have been called, as unmarshalling should fail")
			},
		),
	)

	assert.Error(app.Err())
}

func TestProvideNamed(t *testing.T) {
	t.Run("Success", testProvideNamedSuccess)
	t.Run("Exact", testProvideNamedExact)
}
