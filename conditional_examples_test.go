// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrange

import (
	"fmt"

	"go.uber.org/fx"
)

func ExampleIf() {
	condition := true

	type Config struct {
		Address string
	}

	fx.New(
		fx.NopLogger,
		If(condition).Then(
			fx.Supply(Config{
				Address: ":8080",
			}),
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
