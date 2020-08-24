package arrange

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func ExampleIf() {
	v := viper.New()
	v.Set("address", ":8080")

	type Config struct {
		Address string
	}

	fx.New(
		LoggerWriter(ioutil.Discard),
		Supply(v), // necessary for the Provide call below
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
