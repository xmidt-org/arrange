package arrange

import (
	"reflect"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// Unmarshaler is the strategy used to unmarshal configuration into objects.
// An unnamed fx.App component that implements this interface is required for arrange.
type Unmarshaler interface {
	// Unmarshal reads configuration data into the given struct
	Unmarshal(value interface{}) error

	// UnmarshalKey reads configuration data from a key into the given struct
	UnmarshalKey(key string, value interface{}) error
}

// makeUnmarshalFunc is the common approach to dynamically creating an fx.Provide
// constructor function to unmarshal an object and return the results.  The returned
// functional always has the signature "func(Unmarshaler) (T, error)", where T is the
// pointer type of the component being returned.
//
// The closure passed to this function is expected to do the actual unmarshaling.
func makeUnmarshalFunc(prototype interface{}, uf func(Unmarshaler, interface{}) error) reflect.Value {
	t := NewTarget(prototype)
	return reflect.MakeFunc(
		reflect.FuncOf(
			// function inputs:
			[]reflect.Type{reflect.TypeOf((*Unmarshaler)(nil)).Elem()},

			// function outputs:
			[]reflect.Type{t.Component.Type(), Type[error]()},

			false, // not variadic
		),
		func(args []reflect.Value) []reflect.Value {
			err := uf(
				args[0].Interface().(Unmarshaler),
				t.UnmarshalTo.Interface(),
			)

			return []reflect.Value{
				t.Component,
				NewErrorValue(err),
			}
		},
	)
}

// Unmarshal generates and returns a constructor function that uses the required global
// Unmarshaler component to unmarshal an object.  The unmarshaled object will be the same
// type as the prototype.
//
// Provide is generally preferred to this function, but Unmarshal is more flexible
// and can be used with fx.Annotated.
func Unmarshal(prototype interface{}) interface{} {
	return makeUnmarshalFunc(
		prototype,
		func(u Unmarshaler, value interface{}) error {
			return u.Unmarshal(value)
		},
	).Interface()
}

// UnmarshalKey generates and returns a constructor function that unmarshals an object
// from a specific Viper configuration key.  The object's type will be the same as the prototype.
//
// Generally, ProvideKey is simpler and preferred.  Use this function when more control
// is needed over the component, such as putting it into a group or using a different component name.
func UnmarshalKey(key string, prototype interface{}) interface{} {
	return makeUnmarshalFunc(
		prototype,
		func(u Unmarshaler, value interface{}) error {
			return u.UnmarshalKey(key, value)
		},
	).Interface()
}

// Provide is the simplest way to arrange an unmarshaled component.  This function simply
// returns a constructor that unmarshals a component of the same type as the prototype
// and emits it as an unnamed component.
func Provide(prototype interface{}) fx.Option {
	return fx.Provide(
		Unmarshal(prototype),
	)
}

// ProvideKey arranges for unmarshaling a configuration key into an object of the given
// prototype.  The object is returned as a component with the same name as the key.
func ProvideKey(key string, prototype interface{}, opts ...viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name:   key,
			Target: UnmarshalKey(key, prototype),
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
func (k *K) Provide(prototype interface{}) fx.Option {
	var constructors []interface{}
	for key := range k.keys {
		if len(k.group) > 0 {
			constructors = append(constructors, fx.Annotated{
				Group:  k.group,
				Target: UnmarshalKey(key, prototype),
			})
		} else {
			constructors = append(constructors, fx.Annotated{
				Name:   key,
				Target: UnmarshalKey(key, prototype),
			})
		}
	}

	return fx.Provide(constructors...)
}
