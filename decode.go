package arrange

import (
	"encoding"
	"reflect"

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

// DefaultDecodeHooks is a viper option that sets the decode hooks to more useful defaults.
// This includes the ones set by viper itself, plus hooks defined by this package.
//
// Note that you can still use ComposeDecodeHooks with this option as long as you use
// it after this one.
//
// See https://pkg.go.dev/github.com/spf13/viper#DecodeHook
func DefaultDecodeHooks(dc *mapstructure.DecoderConfig) {
	dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		TextUnmarshalerHookFunc,
	)
}

// ComposeDecodeHooks adds more decode hook functions to mapstructure's DecoderConfig.  If
// there are already decode hooks, they are preserved and the given hooks are appended.
//
// See https://pkg.go.dev/github.com/mitchellh/mapstructure#ComposeDecodeHookFunc
func ComposeDecodeHooks(fs ...mapstructure.DecodeHookFunc) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		if dc.DecodeHook != nil {
			dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
				append([]mapstructure.DecodeHookFunc{dc.DecodeHook},
					fs...,
				)...,
			)
		} else {
			dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(fs...)
		}
	}
}

var (
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// TextUnmarshalerHookFunc is a mapstructure.DecodeHookFunc that honors the destination
// type's encoding.TextUnmarshaler implementation, using it to convert the src.  The src
// parameter must be a string, or else this function does not attempt any conversion.
//
// The to type must be one of two kinds:
//
// First, to can be a non-pointer type which implements encoding.TextUnmarshaler through a
// pointer receiver.  The time.Time type is an example of this.  In this case, this function
// uses reflect.New to create a new instance and invokes UnmarshalText through that pointer.
// The pointer's element is returned, along with any error.
//
// Second, to can be a pointer type which itself implements encoding.TextUnmarshaler.  In this
// case reflect.New is used to create a new instances, then UnmarshalText is invoked through
// that pointer.  That pointer is then returned, along with any error.
//
// This function explicitly does not support more than one level of indirection.  For example,
// **T where *T implements encoding.TextUnmarshaler.
//
// In any case where this function does no conversion, it returns src and a nil error.  This
// is the contract required by mapstructure.DecodeHookFunc.
func TextUnmarshalerHookFunc(_, to reflect.Type, src interface{}) (interface{}, error) {
	if text, ok := src.(string); ok {
		switch {
		// the "to" type is not a pointer and a pointer to "to" implements encoding.TextUnmarshaler
		// this is by far the most common case.  For example:
		//
		// struct {
		//   Time time.Time // non-pointer, but *time.Time implements encoding.TextUnmarshaler
		// }
		case to.Kind() != reflect.Ptr && reflect.PtrTo(to).Implements(textUnmarshalerType):
			ptr := reflect.New(to)
			tu := ptr.Interface().(encoding.TextUnmarshaler)
			err := tu.UnmarshalText([]byte(text))
			return ptr.Elem().Interface(), err

		// the "to" type is a pointer to a value and it implements encoding.TextUnmarshaler
		// commonly occurs with "optional" properties, where a nil value means
		// it wasn't set
		//
		// struct {
		//   Time *time.Time
		// }
		case to.Kind() == reflect.Ptr && to.Elem().Kind() != reflect.Ptr && to.Implements(textUnmarshalerType):
			ptr := reflect.New(to.Elem()) // this will be the same type as "to"
			tu := ptr.Interface().(encoding.TextUnmarshaler)
			err := tu.UnmarshalText([]byte(text))
			return tu, err
		}
	}

	return src, nil
}
