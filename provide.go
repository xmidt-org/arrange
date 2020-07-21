package arrange

import (
	"reflect"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func unmarshalTarget(prototype interface{}) (component, target reflect.Value) {
	pvalue := reflect.ValueOf(prototype)
	if pvalue.Kind() == reflect.Ptr {
		target = reflect.New(pvalue.Type().Elem())
		if !pvalue.IsNil() {
			target.Elem().Set(pvalue.Elem())
		}

		component = target
	} else {
		target = reflect.New(pvalue.Type())
		target.Elem().Set(pvalue)
		component = target.Elem()
	}

	return
}

func unmarshalFuncOf(result reflect.Value) reflect.Type {
	return reflect.FuncOf(
		// inputs:
		[]reflect.Type{reflect.TypeOf(ProvideIn{})},

		// outputs:
		[]reflect.Type{
			result.Type(),
			reflect.TypeOf((*error)(nil)).Elem(),
		},

		// we're not variadic:
		false,
	)
}

type stub func([]reflect.Value) []reflect.Value

func unmarshalStub(component, target reflect.Value, opts ...viper.DecoderConfigOption) stub {
	return func(args []reflect.Value) []reflect.Value {
		var (
			u = args[0].Interface().(ProvideIn)

			err = u.Viper.Unmarshal(
				target.Interface(),
				Merge(u.DecodeOptions, opts),
			)

			errPtr = reflect.New(
				reflect.TypeOf((*error)(nil)).Elem(),
			)
		)

		if err != nil {
			errPtr.Elem().Set(reflect.ValueOf(err))
		}

		return []reflect.Value{
			component,
			errPtr.Elem(),
		}
	}
}

func unmarshalKeyStub(key string, component, target reflect.Value, opts ...viper.DecoderConfigOption) stub {
	return func(args []reflect.Value) []reflect.Value {
		var (
			u = args[0].Interface().(ProvideIn)

			err = u.Viper.UnmarshalKey(
				key,
				target.Interface(),
				Merge(u.DecodeOptions, opts),
			)

			errPtr = reflect.New(
				reflect.TypeOf((*error)(nil)).Elem(),
			)
		)

		if err != nil {
			errPtr.Elem().Set(reflect.ValueOf(err))
		}

		return []reflect.Value{
			component,
			errPtr.Elem(),
		}
	}
}

func unmarshal(prototype interface{}, opts ...viper.DecoderConfigOption) interface{} {
	component, target := unmarshalTarget(prototype)
	return reflect.MakeFunc(
		unmarshalFuncOf(component),
		unmarshalStub(component, target),
	).Interface()
}

func unmarshalKey(key string, prototype interface{}, opts ...viper.DecoderConfigOption) interface{} {
	component, target := unmarshalTarget(prototype)
	return reflect.MakeFunc(
		unmarshalFuncOf(component),
		unmarshalKeyStub(key, component, target),
	).Interface()
}

// Provide is the simplest way to arrange an unmarshaled component.  This function simply
// returns a constructor that unmarshals a component of the same type as the prototype
// and emits it as an unnamed component.
func Provide(prototype interface{}, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		unmarshal(prototype, opts...),
	)
}

// ProvideKey arranges for unmarshaling a configuration key into an object of the given
// prototype.  The object is returned as an unnamed component.
func ProvideKey(key string, prototype interface{}, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		unmarshalKey(key, prototype, opts...),
	)
}

// ProvideNamed arranges for a component to be unmarshaled from a configuration key and
// returned as a named component with the same name as the configuration key.
func ProvideNamed(keyAndName string, prototype interface{}, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   keyAndName,
			Target: unmarshalKey(keyAndName, prototype, opts...),
		},
	)
}
