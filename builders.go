package arrange

import (
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// Optioner is a strategy for creating fx.Option objects.  Most
// of the Fluent Builders in this package implement this interface.
type Optioner interface {
	AppendTo([]fx.Option) []fx.Option
}

// Optioners is a slice type for aggregating Optioner strategies.
// This type implements Optioner itself.
type Optioners []Optioner

// AppendTo adds all Optioner strategies in order to the supplied slice,
// then returns the result
func (o Optioners) AppendTo(options []fx.Option) []fx.Option {
	for _, opt := range o {
		options = opt.AppendTo(options)
	}

	return options
}

// Extend adds more Optioner strategies to this aggregate Optioner
func (o *Optioners) Extend(more ...Optioner) {
	*o = append(*o, more...)
}

// OptionBuilder is the primary, top-level Fluent Builder for arranging unmarshaling.
// When finished building, the Option method must be called to generate the fx.Option
// to place into fx.New.
type OptionBuilder struct {
	o Optioners
}

func Key(key string) *SingleBuilder {
	return new(OptionBuilder).Key(key)
}

func Name(name string) *SingleBuilder {
	return new(OptionBuilder).Name(name)
}

func Group(group string) *SingleBuilder {
	return new(OptionBuilder).Group(group)
}

func Named(first string, rest ...string) *MultiKeyBuilder {
	return new(OptionBuilder).Named(first, rest...)
}

func Unmarshal(prototype interface{}, opts ...viper.DecoderConfigOption) *OptionBuilder {
	return new(OptionBuilder).Unmarshal(prototype, opts...)
}

func (ob *OptionBuilder) Key(key string) *SingleBuilder {
	sb := &SingleBuilder{
		OptionBuilder: ob,
		key:           key,
	}

	ob.o.Extend(sb)
	return sb
}

func (ob *OptionBuilder) Extend(more ...Optioner) *OptionBuilder {
	ob.o.Extend(more...)
	return ob
}

func (ob *OptionBuilder) Name(name string) *SingleBuilder {
	sb := &SingleBuilder{
		OptionBuilder: ob,
		name:          name,
	}

	ob.o.Extend(sb)
	return sb
}

func (ob *OptionBuilder) Group(group string) *SingleBuilder {
	sb := &SingleBuilder{
		OptionBuilder: ob,
		group:         group,
	}

	ob.Extend(sb)
	return sb
}

func (ob *OptionBuilder) Named(first string, rest ...string) *MultiKeyBuilder {
	mb := &MultiKeyBuilder{
		OptionBuilder: ob,
		keys:          append([]string{first}, rest...),
	}

	ob.o.Extend(mb)
	return mb
}

func (ob *OptionBuilder) Unmarshal(prototype interface{}, opts ...viper.DecoderConfigOption) *OptionBuilder {
	sb := &SingleBuilder{
		OptionBuilder: ob,
		prototype:     prototype,
		decodeOptions: opts,
	}

	ob.o.Extend(sb)
	return ob
}

func (ob *OptionBuilder) AppendTo(options []fx.Option) []fx.Option {
	return ob.o.AppendTo(options)
}

func (ob *OptionBuilder) Option() fx.Option {
	return fx.Options(ob.AppendTo(nil)...)
}

type SingleBuilder struct {
	*OptionBuilder
	key           string
	name          string
	group         string
	prototype     interface{}
	decodeOptions []viper.DecoderConfigOption
}

func (sb *SingleBuilder) Name(n string) *SingleBuilder {
	sb.name = n
	sb.group = ""
	return sb
}

func (sb *SingleBuilder) Group(g string) *SingleBuilder {
	sb.name = ""
	sb.group = g
	return sb
}

func (sb *SingleBuilder) Unmarshal(prototype interface{}, opts ...viper.DecoderConfigOption) *OptionBuilder {
	sb.prototype = prototype
	sb.decodeOptions = opts
	return sb.OptionBuilder
}

func (sb *SingleBuilder) Append(options []fx.Option) []fx.Option {
	var constructor interface{}
	if len(sb.key) > 0 {
		constructor = ProvideKey(sb.key, sb.prototype, sb.decodeOptions...)
	} else {
		constructor = Provide(sb.prototype, sb.decodeOptions...)
	}

	if len(sb.name) > 0 {
		options = append(options, fx.Provide(
			fx.Annotated{
				Name:   sb.name,
				Target: constructor,
			},
		))
	} else if len(sb.group) > 0 {
		options = append(options, fx.Provide(
			fx.Annotated{
				Group:  sb.group,
				Target: constructor,
			},
		))
	} else {
		options = append(options, fx.Provide(constructor))
	}

	return options
}

type MultiKeyBuilder struct {
	*OptionBuilder
	keys          []string
	group         string
	prototype     interface{}
	decodeOptions []viper.DecoderConfigOption
}

func (mb *MultiKeyBuilder) Group(g string) *MultiKeyBuilder {
	mb.group = g
	return mb
}

func (mb *MultiKeyBuilder) Unmarshal(prototype interface{}, opts ...viper.DecoderConfigOption) *OptionBuilder {
	mb.prototype = prototype
	mb.decodeOptions = opts
	return mb.OptionBuilder
}

func (mb *MultiKeyBuilder) Append(options []fx.Option) []fx.Option {
	for _, key := range mb.keys {
		constructor := ProvideKey(key, mb.prototype, mb.decodeOptions...)
		if len(mb.group) > 0 {
			options = append(options, fx.Provide(
				fx.Annotated{
					Group:  mb.group,
					Target: constructor,
				},
			))
		} else {
			options = append(options, fx.Provide(
				fx.Annotated{
					Name:   key,
					Target: constructor,
				},
			))
		}
	}

	return options
}
