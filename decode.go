package arrange

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// ErrorUnused sets the DecoderConfig.ErrorUnused flag.  This option
// can be used in place of viper's UnmarshalExact method, as it does
// the exact same thing.
//
// This:
//
//   v := viper.New()
//   v.UnmarshalExact(config)
//
// is the same as this:
//
//   v := viper.New()
//   v.Unmarshal(config, arrange.ErrorUnused(true))
func ErrorUnused(f bool) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.ErrorUnused = f
	}
}

// Exact is a synonym for ErrorUnused(true), which is the most common case.
// Note that the ErrorUnused flag can be turned off individually by doing
// ErrorUnused(false) in subsequent UnmarshalXXX calls.
func Exact(dc *mapstructure.DecoderConfig) {
	dc.ErrorUnused = true
}

// WeaklyTypedInput sets the DecoderConfig.WeaklyTypedInput flag
func WeaklyTypedInput(f bool) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.WeaklyTypedInput = f
	}
}

// TagName sets the DecoderConfig.TagName used when doing reflection
// on struct fields to determine the corresponding configuration keys.
// The default is "mapstructure", and using TagName("") sets that same default.
func TagName(v string) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.TagName = v
	}
}

// Squash sets the DecoderConfig.Squash flag, which affects how embedded
// struct fields are handled
func Squash(f bool) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.Squash = f
	}
}

// Reset returns an option that resets the entire DecoderConfig.
// Use with caution, as this will completely change viper's default behavior.
//
// This options is useful and clearer when a large number of settings need to be changed at once.
func Reset(in mapstructure.DecoderConfig) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		*dc = in
	}
}

// Merge takes any number of slices of decoder options and merges them
// into a single option.
//
// This function avoids consuming more heap to merge slices.  It simply iterates over all
// the given options, applying them in order.
func Merge(opts ...[]viper.DecoderConfigOption) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		for _, group := range opts {
			for _, o := range group {
				o(dc)
			}
		}
	}
}
