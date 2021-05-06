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
		fx.Provide(
			func() []byte {
				// an example of a component created in the normal fashion
				// we'll use this as a suffix, and it needs to be a different
				// type than string since we're binding that type in With
				return []byte("!!!")
			},
		),
		Bind{
			func(v string) {
				// v is supplied by With
				fmt.Println(v)
			},
			func(v string, suffix []byte) *bytes.Buffer {
				// we can use v to construct even more components
				// we can also refer to other components present in the enclosing app
				b := bytes.NewBufferString(v)
				b.Write(suffix)
				return b
			},
			func(b *bytes.Buffer) {
				// ... and we can refer to components created inside this block as desired
				fmt.Println(b.String())
			},
		}.With(info),
		fx.Provide(
			func(b *bytes.Buffer) (int, error) {
				// components created inside Bind.With are normal components
				fmt.Println(b)
				return 123, nil
			},
		),
		fx.Populate(&value),
	)

	fmt.Println(value)

	// Output:
	// here is a lovely little informational string
	// here is a lovely little informational string!!!
	// here is a lovely little informational string!!!
	// 123
}
