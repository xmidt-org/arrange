package arrange

import (
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// ProvideIn is the set of dependencies for all unmarshal providers in this package.
// This set of dependencies is satisified by Supply.
type ProvideIn struct {
	fx.In

	// Viper is the required Viper component in the enclosing fx.App
	Viper *viper.Viper

	// DecoderOptions are an optional set of options from the enclosing fx.App
	DecoderOptions []viper.DecoderConfigOption `optional:"true"`

	// Printer is an optional fx.Printer that arrange will use to output informational messages
	Printer fx.Printer `optional:"true"`
}

// Supply is an analog to fx.Supply.  This eases the injection of a viper instance
// into an fx.App.  If the viper instance is nil, an fx.Error option is used to short-circuit
// the app startup.  If no options are supplied, no component for the options is provided.
//
// Use of this function is entirely optional.  You can use fx.Supply instead.  This function
// just handles the nil viper case gracefully and makes supplying options a bit easier.
func Supply(v *viper.Viper, opts ...viper.DecoderConfigOption) fx.Option {
	if v == nil {
		return fx.Error(ErrNilViper)
	}

	if len(opts) > 0 {
		return fx.Supply(v, opts)
	}

	return fx.Supply(v)
}
