package arrange

import (
	"reflect"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// unmarshal is the common approach to dynamically creating an fx.Provide
// constructor function to unmarshal an object and return the results
//
// the closure passed to this function is expected to handle unmarshaling.
// the global decoder options are passed, and the closure can merge them
// with any local options.
func unmarshal(prototype interface{}, uf func(*viper.Viper, []viper.DecoderConfigOption, interface{}) error) interface{} {
	t := NewTarget(prototype)
	return reflect.MakeFunc(
		reflect.FuncOf(
			// function inputs:
			[]reflect.Type{reflect.TypeOf(ProvideIn{})},

			// function outputs:
			[]reflect.Type{t.ComponentType(), ErrorType()},

			false, // not variadic
		),
		func(args []reflect.Value) []reflect.Value {
			in := args[0].Interface().(ProvideIn)
			err := uf(in.Viper, in.DecoderOptions, t.unmarshalTo.Interface())
			return []reflect.Value{
				t.component,
				NewErrorValue(err),
			}
		},
	).Interface()

}

// Unmarshal generates and returns a constructor function that unmarshals an object from
// Viper.  The object's type will be the same as the prototype.
//
// Provide is generally preferred to this function, but Unmarshal is more flexible
// and can be used with fx.Annotated.
func Unmarshal(prototype interface{}, local ...viper.DecoderConfigOption) interface{} {
	return unmarshal(
		prototype,
		func(v *viper.Viper, global []viper.DecoderConfigOption, c interface{}) error {
			return v.Unmarshal(c, Merge(global, local))
		},
	)
}

// UnmarshalKey generates and returns a constructor function that unmarshals an object
// from a specific Viper configuration key.  The object's type will be the same as the prototype.
//
// Generally, ProvideKey is simpler and preferred.  Use this function when more control
// is needed over the component, such as putting it into a group or using a different component name.
func UnmarshalKey(key string, prototype interface{}, local ...viper.DecoderConfigOption) interface{} {
	return unmarshal(
		prototype,
		func(v *viper.Viper, global []viper.DecoderConfigOption, c interface{}) error {
			return v.UnmarshalKey(key, c, Merge(global, local))
		},
	)
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
