package arrange

import (
	"errors"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var (
	// ErrNilViper is returned to the fx.App when the externally supplied Viper
	// instance is nil
	ErrNilViper = errors.New("the viper instance cannot be nil")
)

// Unmarshaler is the strategy used to unmarshal configuration into objects.
// An unnamed fx.App component that implements this interface is required for arrange.
type Unmarshaler interface {
	// Unmarshal reads configuration data into the given struct
	Unmarshal(value interface{}) error

	// UnmarshalKey reads configuration data from a key into the given struct
	UnmarshalKey(key string, value interface{}) error
}

// ViperUnmarshaler is the standard Unmarshaler implementation used by arrange.
// It couples a Viper instance together with zero or more decoder options.
type ViperUnmarshaler struct {
	// Viper is the required Viper instance to which all unmarshal operations are delegated
	Viper *viper.Viper

	// Options is the optional slice of viper.DecoderConfigOptions passed to all
	// unmarshal calls
	Options []viper.DecoderConfigOption

	// Printer is the required fx.Printer component to which informational messages are written
	Printer fx.Printer
}

// Unmarshal implements Unmarshaler
func (vu ViperUnmarshaler) Unmarshal(value interface{}) error {
	vu.Printer.Printf("UNMARSHAL => %T", value)
	return vu.Viper.Unmarshal(value, vu.Options...)
}

// UnmarshalKey implements Unmarshaler
func (vu ViperUnmarshaler) UnmarshalKey(key string, value interface{}) error {
	vu.Printer.Printf("UNMARSHAL KEY\t[%s] => %T", key, value)
	return vu.Viper.UnmarshalKey(key, value, vu.Options...)
}

// ViperUnmarshalIn is the set of dependencies required to build a ViperUnmarshaler
type ViperUnmarshalerIn struct {
	fx.In

	// Viper is the required viper instance
	Viper *viper.Viper

	// Options is the optional slice of viper.DecoderConfigOption that will be
	// applied to every unmarshal or unmarshal key operation
	Options []viper.DecoderConfigOption `optional:"true"`

	// Printer is an optional fx.Printer component to which the viper unmarshaler
	// prints informational messages.  If not supplied, DefaultPrinter() is used.
	Printer fx.Printer `optional:"true"`
}

// ForViper supplies an externally created Viper instance to the enclosing ex.App.
// This function also creates an Unmarshaler component backed by this Viper instance.
//
// The set of viper.DecoderConfigOptions used will be the merging of the options supplied
// to this function and an optional []viper.DecoderConfigOption component.
func ForViper(v *viper.Viper, o ...viper.DecoderConfigOption) fx.Option {
	if v == nil {
		return fx.Error(ErrNilViper)
	}

	return fx.Options(
		fx.Supply(v),
		fx.Provide(
			func(in ViperUnmarshalerIn) Unmarshaler {
				return ViperUnmarshaler{
					Viper: in.Viper,
					Options: append(
						append([]viper.DecoderConfigOption{}, o...),
						in.Options...,
					),
					Printer: NewModulePrinter(Module, in.Printer),
				}
			},
		),
	)
}
