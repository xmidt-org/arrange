package arrange

import (
	"reflect"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// UnmarshalIn is the set of dependencies for all UnmarshalXXX functions in this package
type UnmarshalIn struct {
	fx.In

	// Viper is the required Viper component in the enclosing fx.App
	Viper *viper.Viper

	// DecodeOptions are an optional set of options from the enclosing fx.App
	DecodeOptions []viper.DecoderConfigOption `optional:"true"`
}

// unmarshalProvider is the strategy type used to emit unmarshaled components
// into an fx.App
type unmarshalProvider struct {
	key           string
	exact         bool
	component     reflect.Value
	target        reflect.Value
	decodeOptions []viper.DecoderConfigOption
}

// unmarshal performs the actual unmarshaling, using the function signature
// expected by reflect.MakeFunc
func (up unmarshalProvider) unmarshal(args []reflect.Value) []reflect.Value {
	u := args[0].Interface().(UnmarshalIn)
	var err error
	switch {
	case len(up.key) > 0:
		err = u.Viper.UnmarshalKey(
			up.key,
			up.target.Interface(),
			Merge(u.DecodeOptions, up.decodeOptions),
		)

	case up.exact:
		err = u.Viper.UnmarshalExact(
			up.target.Interface(),
			Merge(u.DecodeOptions, up.decodeOptions),
		)

	default:
		err = u.Viper.Unmarshal(
			up.target.Interface(),
			Merge(u.DecodeOptions, up.decodeOptions),
		)
	}

	errPtr := reflect.New(
		reflect.TypeOf((*error)(nil)).Elem(),
	)

	if err != nil {
		errPtr.Elem().Set(reflect.ValueOf(err))
	}

	return []reflect.Value{
		up.component,
		errPtr.Elem(),
	}
}

// provide creates the actual constructor function that unmarshals
// the appropriate type
func (up unmarshalProvider) provide() interface{} {
	return reflect.MakeFunc(
		// the function type, which is the signature of unmarshal
		reflect.FuncOf(
			// inputs:
			[]reflect.Type{reflect.TypeOf(UnmarshalIn{})},

			// outputs:
			[]reflect.Type{
				up.component.Type(),
				reflect.TypeOf((*error)(nil)).Elem(),
			},

			// we're not variadic:
			false,
		),
		up.unmarshal,
	).Interface()
}

// newUnmarshalProvider initializes an unmarshalProvider from a prototype object
func newUnmarshalProvider(prototype interface{}, opts ...viper.DecoderConfigOption) (up unmarshalProvider) {
	up.decodeOptions = opts

	pvalue := reflect.ValueOf(prototype)
	if pvalue.Kind() == reflect.Ptr {
		up.target = reflect.New(pvalue.Type().Elem())
		if !pvalue.IsNil() {
			up.target.Elem().Set(pvalue.Elem())
		}

		up.component = up.target
	} else {
		up.target = reflect.New(pvalue.Type())
		up.target.Elem().Set(pvalue)
		up.component = up.target.Elem()
	}

	return
}

// Unmarshal returns an provider function that produces an object that has been unmarshaled
// from a viper instance.  The viper instance must be a component, often supplied by fx.Supply.
//
// See: https://pkg.go.dev/github.com/spf13/viper?tab=doc#Unmarshal
//
// If prototype is a pointer type, the component will be a pointer of the same type.  If prototype
// is a non-nil pointer, the object pointed to will be used as the default value.
//
// If prototype is not a pointer, the component will be of the same concrete type.  The prototype
// will be copied as the default value.
//
// For example:
//
//   v := viper.New() // more initialization not shown
//   fx.App(
//     fx.Supply(v),
//
//     // NOTE: Remember that only (1) instance of a type can exist in an fx.App.
//     // Below are just some examples of various ways to use Unmarshal.
//
//     fx.Provide(
//       // creates a component of type Config that has 23 as the initial value for the Age field
//       arrange.Unmarshal(Config{Age: 23}),
//
//       // creates a component of type *Config that has no defaults
//       arrange.Unmarshal((*Config)(nil)),
//
//       // creates a component of type *Config that has 23 as the initial value for the Age field
//       arrange.Unmarshal(&Config{Age: 23}),
//
//       // creates a named component of type Config that is unmarshaled
//       fx.Annotated{
//         Name: "config",
//         Target: arrange.Unmarshal(Config{}),
//       },
//     ),
//   )
func Unmarshal(prototype interface{}, opts ...viper.DecoderConfigOption) interface{} {
	return newUnmarshalProvider(prototype, opts...).provide()
}

// UnmarshalKey uses viper.UnmarshalKey to unmarshal its component, but is in
// all other ways similar to Unmarshal.
//
// See: https://pkg.go.dev/github.com/spf13/viper?tab=doc#UnmarshalKey
//
// Typically, this function is used when there are multiple components of the same type
// being unmarshaled from different keys.  To handle that case, use fx.Annotated:
//
//   v := viper.New() // more initialization not shown
//   fx.App(
//     fx.Supply(v),
//     fx.Provide(
//       fx.Annotated{
//         Name: "foo", // can be any name you like
//         Target: arrange.UnmarshalKey("server.main", Config{}),
//       },
//
//       fx.Annotated{
//         Name: "bar", // can be any name you like
//         Target: arrange.UnmarshalKey("server.health", Config{}),
//       },
//
//       fx.Annotated{
//         Group: "configs", // groups work just fine too
//         Target: arrange.UnmarshalKey("server.another", Config{}),
//       },
//     ),
//   )
func UnmarshalKey(key string, prototype interface{}, opts ...viper.DecoderConfigOption) interface{} {
	up := newUnmarshalProvider(prototype, opts...)
	up.key = key
	return up.provide()
}

// UnmarshalNamed is syntactic sugar for a very common use case:  unmarshaling a component from a key
// and naming that component the same as the key.
//
// This:
//
//   v := viper.New()
//   fx.App(
//     fx.Supply(v),
//     fx.Provide(
//       fx.Annotated{
//         Name: "server.main",
//         Target: arrange.UnmarshalKey("server.main", Config{}),
//       },
//     ),
//   )
//
// is the same as:
//
//   v := viper.New()
//   fx.App(
//     fx.Supply(v),
//     fx.Provide(
//       arrange.UnmarshalNamed("server.main", Config{}),
//     ),
//   )
func UnmarshalNamed(keyAndName string, prototype interface{}, opts ...viper.DecoderConfigOption) fx.Annotated {
	return fx.Annotated{
		Name:   keyAndName,
		Target: UnmarshalKey(keyAndName, prototype, opts...),
	}
}
