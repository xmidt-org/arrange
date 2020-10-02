package arrange

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorUnused(t *testing.T) {
	var (
		assert = assert.New(t)
		dc     mapstructure.DecoderConfig
	)

	ErrorUnused(true)(&dc)
	assert.True(dc.ErrorUnused)

	ErrorUnused(false)(&dc)
	assert.False(dc.ErrorUnused)
}

func TestExact(t *testing.T) {
	var (
		assert = assert.New(t)
		dc     mapstructure.DecoderConfig

		o viper.DecoderConfigOption = Exact
	)

	o(&dc)
	assert.True(dc.ErrorUnused)
}

func TestWeaklyTypedInput(t *testing.T) {
	var (
		assert = assert.New(t)
		dc     mapstructure.DecoderConfig
	)

	WeaklyTypedInput(true)(&dc)
	assert.True(dc.WeaklyTypedInput)

	WeaklyTypedInput(false)(&dc)
	assert.False(dc.WeaklyTypedInput)
}

func TestTagName(t *testing.T) {
	var (
		assert = assert.New(t)
		dc     mapstructure.DecoderConfig
	)

	TagName("tag1")(&dc)
	assert.Equal("tag1", dc.TagName)

	TagName("")(&dc)
	assert.Equal("", dc.TagName)

	TagName("tag2")(&dc)
	assert.Equal("tag2", dc.TagName)
}

func TestSquash(t *testing.T) {
	var (
		assert = assert.New(t)
		dc     mapstructure.DecoderConfig
	)

	Squash(true)(&dc)
	assert.True(dc.Squash)

	Squash(false)(&dc)
	assert.False(dc.Squash)
}

func TestReset(t *testing.T) {
	var (
		assert = assert.New(t)
		dc     mapstructure.DecoderConfig
	)

	Reset(mapstructure.DecoderConfig{
		Squash:  true,
		TagName: "test1",
	})(&dc)

	assert.Equal(
		mapstructure.DecoderConfig{
			Squash:  true,
			TagName: "test1",
		},
		dc,
	)

	Reset(mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		ErrorUnused:      true,
		TagName:          "",
	})(&dc)

	assert.Equal(
		mapstructure.DecoderConfig{
			WeaklyTypedInput: true,
			ErrorUnused:      true,
			TagName:          "",
		},
		dc,
	)
}

func TestMerge(t *testing.T) {
	var (
		assert = assert.New(t)
		dc     mapstructure.DecoderConfig
	)

	Merge(
		[]viper.DecoderConfigOption{
			Exact,
		},
	)(&dc)

	assert.Equal(
		mapstructure.DecoderConfig{
			ErrorUnused: true,
		},
		dc,
	)

	Merge(
		[]viper.DecoderConfigOption{
			Reset(mapstructure.DecoderConfig{}),
			Squash(true),
		},
		[]viper.DecoderConfigOption{
			Squash(false),
		},
		[]viper.DecoderConfigOption{
			WeaklyTypedInput(true),
			ErrorUnused(true),
		},
	)(&dc)

	assert.Equal(
		mapstructure.DecoderConfig{
			WeaklyTypedInput: true,
			ErrorUnused:      true,
		},
		dc,
	)
}

func TestDefaultDecodeHooks(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		config mapstructure.DecoderConfig
	)

	const timeString = "1998-11-13T12:11:56Z"

	expectedTime, err := time.Parse(time.RFC3339, timeString)
	require.NoError(err)

	DefaultDecodeHooks(&config)
	require.NotNil(config.DecodeHook)

	result, err := mapstructure.DecodeHookExec(
		config.DecodeHook,
		reflect.TypeOf(""),
		reflect.TypeOf(time.Duration(0)),
		"15s",
	)

	assert.Equal(15*time.Second, result)
	assert.NoError(err)

	result, err = mapstructure.DecodeHookExec(
		config.DecodeHook,
		reflect.TypeOf(""),
		reflect.TypeOf([]string{}),
		"a,b,c",
	)

	assert.Equal([]string{"a", "b", "c"}, result)
	assert.NoError(err)

	result, err = mapstructure.DecodeHookExec(
		config.DecodeHook,
		reflect.TypeOf(""),
		reflect.TypeOf(time.Time{}),
		timeString,
	)

	assert.Equal(expectedTime, result)
	assert.NoError(err)
}

func testComposeDecodeHooksInitiallyNil(t *testing.T) {
	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				hooks         []mapstructure.DecodeHookFunc
				expectedOrder []int
				actualOrder   []int
				config        mapstructure.DecoderConfig
			)

			for i := 0; i < length; i++ {
				i := i
				expectedOrder = append(expectedOrder, i)
				hooks = append(hooks, func(from, to reflect.Type, src interface{}) (interface{}, error) {
					assert.Equal(reflect.TypeOf(""), from)
					assert.Equal(reflect.TypeOf(int(0)), to)
					assert.Equal("test", src)
					actualOrder = append(actualOrder, i)
					return src, nil
				})
			}

			ComposeDecodeHooks(hooks...)(&config)
			require.NotNil(config.DecodeHook)

			mapstructure.DecodeHookExec(
				config.DecodeHook,
				reflect.TypeOf(""),
				reflect.TypeOf(int(0)),
				"test",
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func testComposeDecodeHooksAppendToExisting(t *testing.T) {
	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				hooks         []mapstructure.DecodeHookFunc
				expectedOrder = []int{0}
				actualOrder   []int
				config        = mapstructure.DecoderConfig{
					DecodeHook: func(from, to reflect.Type, src interface{}) (interface{}, error) {
						assert.Equal(reflect.TypeOf(""), from)
						assert.Equal(reflect.TypeOf(int(0)), to)
						assert.Equal("test", src)
						actualOrder = append(actualOrder, 0)
						return src, nil
					},
				}
			)

			for i := 0; i < length; i++ {
				i := i
				expectedOrder = append(expectedOrder, i+1)
				hooks = append(hooks, func(from, to reflect.Type, src interface{}) (interface{}, error) {
					assert.Equal(reflect.TypeOf(""), from)
					assert.Equal(reflect.TypeOf(int(0)), to)
					assert.Equal("test", src)
					actualOrder = append(actualOrder, i+1)
					return src, nil
				})
			}

			ComposeDecodeHooks(hooks...)(&config)
			require.NotNil(config.DecodeHook)

			mapstructure.DecodeHookExec(
				config.DecodeHook,
				reflect.TypeOf(""),
				reflect.TypeOf(int(0)),
				"test",
			)

			assert.Equal(expectedOrder, actualOrder)
		})
	}
}

func TestComposeDecodeHooks(t *testing.T) {
	t.Run("InitiallyNil", testComposeDecodeHooksInitiallyNil)
	t.Run("AppendToExisting", testComposeDecodeHooksAppendToExisting)
}

func TestTextUnmarshalerHookFunc(t *testing.T) {
	const timeString = "2013-07-11T09:13:07Z"

	expectedTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		t.Fatal(err)
	}

	var (
		testData = []struct {
			from reflect.Type
			to   reflect.Type
			src  interface{}

			expected   interface{}
			expectsErr bool
		}{
			{
				from:     reflect.TypeOf(int(0)),
				to:       reflect.TypeOf(""),
				src:      123,
				expected: 123,
			},
			{
				from:     reflect.TypeOf(""),
				to:       reflect.TypeOf(time.Time{}),
				src:      timeString,
				expected: expectedTime,
			},
			{
				from:     reflect.TypeOf(""),
				to:       reflect.TypeOf(new(time.Time)),
				src:      timeString,
				expected: &expectedTime,
			},
		}
	)

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert            = assert.New(t)
				actual, actualErr = TextUnmarshalerHookFunc(record.from, record.to, record.src)
			)

			assert.Equal(record.expected, actual)
			assert.Equal(record.expectsErr, actualErr != nil)
		})
	}
}
