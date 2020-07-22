package arrange

import (
	"fmt"
	"io/ioutil"
	"log"

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
		fx.Logger(log.New(ioutil.Discard, "", 0)),
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
