package arrange

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func ExampleIf() {
	v := viper.New()
	v.Set("address", ":8080")

	type Config struct {
		Address string
	}

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: zap.NewNop()}
		}),
		ForViper(v), // necessary for the Provide call below
		If(v.IsSet("address")).Then(
			Provide(Config{}),
			fx.Invoke(
				func(cfg Config) error {
					fmt.Println("address", cfg.Address)
					return nil
				},
			),
		),
	)

	// Output:
	// address :8080
}
