package arrange

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func ExampleProvide() {
	const json = `{
		"address": ":8080",
		"timeout": "15s"
	}`

	v := viper.New()
	v.SetConfigType("json")
	v.ReadConfig(strings.NewReader(json))

	type Config struct {
		Address string
		Timeout time.Duration
	}

	fx.New(
		fx.Logger(log.New(ioutil.Discard, "", 0)),
		Supply(v),
		Provide(Config{}),
		fx.Invoke(
			func(cfg Config) error {
				fmt.Println("address", cfg.Address, "timeout", cfg.Timeout)
				return nil
			},
		),
	)

	// Output:
	// address :8080 timeout 15s
}

func ExampleProvideKey() {
	const json = `{
		"server": {
			"address": ":8080",
			"timeout": "15s"
		}
	}`

	v := viper.New()
	v.SetConfigType("json")
	v.ReadConfig(strings.NewReader(json))

	type Config struct {
		Address string
		Timeout time.Duration
	}

	type ConfigIn struct {
		fx.In
		Config Config `name:"server"`
	}

	fx.New(
		fx.Logger(log.New(ioutil.Discard, "", 0)),
		Supply(v),
		ProvideKey("server", Config{}),
		fx.Invoke(
			func(in ConfigIn) error {
				fmt.Println("address", in.Config.Address, "timeout", in.Config.Timeout)
				return nil
			},
		),
	)

	// Output:
	// address :8080 timeout 15s
}

func ExampleKeys() {
	const json = `{
		"servers": {
			"http": {
				"address": ":8080",
				"timeout": "15s"
			},
			"pprof": {
				"address": "localhost:9999"
			}
		}
	}`

	v := viper.New()
	v.SetConfigType("json")
	v.ReadConfig(strings.NewReader(json))

	type Config struct {
		Address string
		Timeout time.Duration
	}

	type ConfigIn struct {
		fx.In
		Http  Config `name:"servers.http"`
		Pprof Config `name:"servers.pprof"`
	}

	fx.New(
		fx.Logger(log.New(ioutil.Discard, "", 0)),
		Supply(v),
		Keys("servers.http", "servers.pprof").Provide(Config{}),
		fx.Invoke(
			func(in ConfigIn) error {
				fmt.Println("http", "address", in.Http.Address, "timeout", in.Http.Timeout)
				fmt.Println("pprof", "address", in.Pprof.Address, "timeout", in.Pprof.Timeout)
				return nil
			},
		),
	)

	// Output:
	// http address :8080 timeout 15s
	// pprof address localhost:9999 timeout 0s
}
