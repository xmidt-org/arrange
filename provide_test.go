package arrange

import (
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func testUnmarshalSuccess(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()

		actual TestConfig
	)

	v.Set("name", "first")
	v.Set("age", 1)

	fxtest.New(
		t,
		Supply(v),
		fx.Provide(
			Unmarshal(TestConfig{}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "first", Age: 1},
		actual,
	)
}

func testUnmarshalExact(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()

		globalCalled = false
		global       = func(*mapstructure.DecoderConfig) {
			globalCalled = true
		}

		actual TestConfig
	)

	v.Set("name", "first")
	v.Set("age", 1)
	v.Set("nosuch", "asdfasdfasdf")

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		TestLogger(t),
		Supply(v, global),
		fx.Provide(
			Unmarshal(TestConfig{}, Exact),
		),
		fx.Populate(&actual),
	)

	assert.True(globalCalled)
	assert.Error(app.Err())
}

func TestUnmarshal(t *testing.T) {
	t.Run("Success", testUnmarshalSuccess)
	t.Run("Exact", testUnmarshalExact)
}

func testUnmarshalKeySuccess(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()

		actual TestConfig
	)

	v.Set("test.name", "first")
	v.Set("test.age", 1)

	fxtest.New(
		t,
		Supply(v),
		fx.Provide(
			UnmarshalKey("test", TestConfig{}),
		),
		fx.Populate(&actual),
	)

	assert.Equal(
		TestConfig{Name: "first", Age: 1},
		actual,
	)
}

func testUnmarshalKeyExact(t *testing.T) {
	var (
		assert = assert.New(t)
		v      = viper.New()

		globalCalled = false
		global       = func(*mapstructure.DecoderConfig) {
			globalCalled = true
		}

		actual TestConfig
	)

	v.Set("test.name", "first")
	v.Set("test.age", 1)
	v.Set("test.nosuch", "asdfasdfasdf")

	t.Log("EXPECTED ERROR OUTPUT:")

	app := fx.New(
		TestLogger(t),
		Supply(v, global),
		fx.Provide(
			UnmarshalKey("test", TestConfig{}, Exact),
		),
		fx.Populate(&actual),
	)

	assert.True(globalCalled)
	assert.Error(app.Err())
}

func TestUnmarshalKey(t *testing.T) {
	t.Run("Success", testUnmarshalKeySuccess)
	t.Run("Exact", testUnmarshalKeyExact)
}

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
		Provide(&AnotherConfig{Interval: 17 * time.Hour}),
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
		AnotherConfig{Name: "testy mctest", Age: 64, Interval: 17 * time.Hour},
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
		TestLogger(t),
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
test1:
  name: "first"
  age: 1
test2:
  name: "second"
  age: 2
test3:
  name: "third"
  age: 3
`

	type In struct {
		fx.In

		Test1 TestConfig  `name:"test1"`
		Test2 *TestConfig `name:"test2"`
		Test3 *TestConfig `name:"test3"`
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		actual In
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		Supply(v),
		ProvideKey("test1", TestConfig{Interval: 15 * time.Second}),
		ProvideKey("test2", &TestConfig{Interval: 23 * time.Minute}),
		ProvideKey("test3", (*TestConfig)(nil)),
		fx.Invoke(
			func(in In) {
				actual = in
			},
		),
	)

	assert.Equal(
		TestConfig{Name: "first", Age: 1, Interval: 15 * time.Second},
		actual.Test1,
	)

	require.NotNil(actual.Test2)
	assert.Equal(
		TestConfig{Name: "second", Age: 2, Interval: 23 * time.Minute},
		*actual.Test2,
	)

	require.NotNil(actual.Test3)
	assert.Equal(
		TestConfig{Name: "third", Age: 3},
		*actual.Test3,
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
		TestLogger(t),
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

func TestKeys(t *testing.T) {
	const yaml = `
test1:
  name: first
  age: 11
test2:
  name: second
  age: 22
test3:
  name: third
  age: 33
`
	type In struct {
		fx.In

		Test1 TestConfig `name:"test1"`
		Test2 TestConfig `name:"test2"`
		Test3 TestConfig `name:"test3"`

		Tests []*TestConfig `group:"tests"`
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()

		globalCount int
		global      = func(*mapstructure.DecoderConfig) {
			globalCount++
		}

		option1Count int
		option1      = func(*mapstructure.DecoderConfig) {
			option1Count++
		}

		option2Count int
		option2      = func(*mapstructure.DecoderConfig) {
			option2Count++
		}

		actual In
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		Supply(v, global),
		Keys("test1", "test2", "test3").Provide(TestConfig{}, option1),
		Keys("test1", "test2", "test3").Group("tests").Provide(&TestConfig{}, option2),
		fx.Invoke(
			func(in In) {
				actual = in
			},
		),
	)

	assert.Equal(
		TestConfig{Name: "first", Age: 11},
		actual.Test1,
	)

	assert.Equal(
		TestConfig{Name: "second", Age: 22},
		actual.Test2,
	)

	assert.Equal(
		TestConfig{Name: "third", Age: 33},
		actual.Test3,
	)

	assert.ElementsMatch(
		[]*TestConfig{
			&TestConfig{Name: "first", Age: 11},
			&TestConfig{Name: "second", Age: 22},
			&TestConfig{Name: "third", Age: 33},
		},
		actual.Tests,
	)

	// called once per unmarshal ...
	assert.Equal(6, globalCount)
	assert.Equal(3, option1Count)
	assert.Equal(3, option2Count)
}
