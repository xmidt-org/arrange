package arrange

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
