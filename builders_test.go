package arrange

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func testOptionBuilderSingle(t *testing.T) {
	const yaml = `
name: "toplevel"
age: 67

key1:
  name: "nested1"
  age: 42

key2:
  name: "nested2"
  age: 90

named:
  name: "nested3"
  age: 13
`

	type In struct {
		fx.In

		TopLevel TestConfig
		Key1     *TestConfig
		Key2     *TestConfig `name:"second"`
		Named    *TestConfig `name:"named"`
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)

		v      = viper.New()
		actual In
	)

	v.SetConfigType("yaml")
	require.NoError(v.ReadConfig(strings.NewReader(yaml)))

	fxtest.New(
		t,
		Supply(v),
		Unmarshal(TestConfig{}).
			Key("key1").Unmarshal(&TestConfig{}).
			Key("key2").Name("second").Unmarshal(&TestConfig{}).
			Named("named").Unmarshal((*TestConfig)(nil)).
			Option(),
		fx.Invoke(
			func(in In) {
				actual = in
			},
		),
	)

	assert.Equal(
		TestConfig{Name: "toplevel", Age: 67},
		actual.TopLevel,
	)

	require.NotNil(actual.Key1)
	assert.Equal(
		TestConfig{Name: "nested1", Age: 42},
		*actual.Key1,
	)

	require.NotNil(actual.Key2)
	assert.Equal(
		TestConfig{Name: "nested2", Age: 90},
		*actual.Key2,
	)

	require.NotNil(actual.Named)
	assert.Equal(
		TestConfig{Name: "nested3", Age: 13},
		*actual.Named,
	)
}

func TestOptionBuilder(t *testing.T) {
	t.Run("Single", testOptionBuilderSingle)
}
