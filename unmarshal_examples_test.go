package arrange

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
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
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: zap.NewNop()}
		}),
		ForViper(v),
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
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: zap.NewNop()}
		}),
		ForViper(v),
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
		HTTP  Config `name:"servers.http"`
		Pprof Config `name:"servers.pprof"`
	}

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: zap.NewNop()}
		}),
		ForViper(v),
		Keys("servers.http", "servers.pprof").Provide(Config{}),
		fx.Invoke(
			func(in ConfigIn) error {
				fmt.Println("http", "address", in.HTTP.Address, "timeout", in.HTTP.Timeout)
				fmt.Println("pprof", "address", in.Pprof.Address, "timeout", in.Pprof.Timeout)
				return nil
			},
		),
	)

	// Output:
	// http address :8080 timeout 15s
	// pprof address localhost:9999 timeout 0s
}
