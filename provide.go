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

// Unmarshal generates and returns a constructor function that unmarshals an object from
// Viper.  The object's type will be the same as the prototype.
//
// Provide is generally preferred to this function, but Unmarshal is more flexible
// and can be used with fx.Annotated.
func Unmarshal(prototype interface{}, opts ...viper.DecoderConfigOption) interface{} {
	component, target := unmarshalTarget(prototype)
	return reflect.MakeFunc(
		unmarshalFuncOf(component),
		unmarshalStub(component, target, opts...),
	).Interface()
}

// UnmarshalKey generates and returns a constructor function that unmarshals an object
// from a specific Viper configuration key.  The object's type will be the same as the prototype.
//
// Generally, ProvideKey is simpler and preferred.  Use this function when more control
// is needed over the component, such as putting it into a group or using a different component name.
func UnmarshalKey(key string, prototype interface{}, opts ...viper.DecoderConfigOption) interface{} {
	component, target := unmarshalTarget(prototype)
	return reflect.MakeFunc(
		unmarshalFuncOf(component),
		unmarshalKeyStub(key, component, target, opts...),
	).Interface()
}

// Provide is the simplest way to arrange an unmarshaled component.  This function simply
// returns a constructor that unmarshals a component of the same type as the prototype
// and emits it as an unnamed component.
func Provide(prototype interface{}, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		Unmarshal(prototype, opts...),
	)
}

// ProvideKey arranges for unmarshaling a configuration key into an object of the given
// prototype.  The object is returned as a component with the same name as the key.
func ProvideKey(key string, prototype interface{}, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: UnmarshalKey(key, prototype, opts...),
		},
	)
}

// K is a simple builder for unmarshaling several Viper keys to the same type
type K struct {
	group string
	keys  map[string]bool
}

// Keys starts a builder chain for unmarshaling several Viper keys to the same type.
// The objects will either be named the same as their keys, or placed within a group
// if the Group method is called with a non-empty string.
func Keys(values ...string) *K {
	k := &K{
		keys: make(map[string]bool, len(values)),
	}

	for _, v := range values {
		k.keys[v] = true
	}

	return k
}

// Group switches the provide logic to place all unmarshaled objects under
// this specific group.  If this method is not called, each object is placed
// as a named component with the same name as its key.
func (k *K) Group(g string) *K {
	k.group = g
	return k
}

// Provide returns an fx.Option that unmarshals all the keys, either named individually
// or under a single group.
func (k *K) Provide(prototype interface{}, opts ...viper.DecoderConfigOption) fx.Option {
	var constructors []interface{}
	for key := range k.keys {
		if len(k.group) > 0 {
			constructors = append(constructors, fx.Annotated{
				Group:  k.group,
				Target: UnmarshalKey(key, prototype, opts...),
			})
		} else {
			constructors = append(constructors, fx.Annotated{
				Name:   key,
				Target: UnmarshalKey(key, prototype, opts...),
			})
		}
	}

	return fx.Provide(constructors...)
}
