package arrange

import (
	"bytes"
	"fmt"

	"go.uber.org/fx"
)

func ExampleBind() {
	info := "here is a lovely little informational string"
	var value int

	fx.New(
		DiscardLogger(),
		Bind{
			func(v string) {
				// v is supplied by With
				fmt.Println(v)
			},
			func(v string) *bytes.Buffer {
				// we can use v to construct even more components
				return bytes.NewBufferString(v)
			},
			func(b *bytes.Buffer) {
				// ... and we can refer to those components as desired elsewhere
				fmt.Println(b.String())
			},
		}.With(info),
		fx.Provide(
			func(unused *bytes.Buffer) (int, error) {
				// components created inside Bind.With are normal components
				return 123, nil
			},
		),
		fx.Populate(&value),
	)

	fmt.Println(value)

	// Output:
	// here is a lovely little informational string
	// here is a lovely little informational string
	// 123
}
