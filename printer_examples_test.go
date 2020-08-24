package arrange

import (
	"fmt"
	"os"

	"go.uber.org/fx"
)

func ExampleLoggerWriter() {
	var component string

	fx.New(
		LoggerWriter(os.Stdout),
		fx.Supply("component"),
		fx.Populate(&component),
	)

	// Output:
	// [Fx] PROVIDE	fx.Printer <= github.com/xmidt-org/arrange.Logger.func1()
	// [Fx] SUPPLY	string
	// [Fx] PROVIDE	fx.Lifecycle <= go.uber.org/fx.New.func1()
	// [Fx] PROVIDE	fx.Shutdowner <= go.uber.org/fx.(*App).shutdowner-fm()
	// [Fx] PROVIDE	fx.DotGraph <= go.uber.org/fx.(*App).dotGraph-fm()
	// [Fx] INVOKE		reflect.makeFuncStub()
}

func ExampleLoggerFunc() {
	var component string

	fx.New(
		LoggerFunc(
			func(template string, args ...interface{}) {
				fmt.Printf(template+"\n", args...)
			},
		),
		fx.Supply("component"),
		fx.Populate(&component),
	)

	// Output:
	// [Fx] PROVIDE	fx.Printer <= github.com/xmidt-org/arrange.Logger.func1()
	// [Fx] SUPPLY	string
	// [Fx] PROVIDE	fx.Lifecycle <= go.uber.org/fx.New.func1()
	// [Fx] PROVIDE	fx.Shutdowner <= go.uber.org/fx.(*App).shutdowner-fm()
	// [Fx] PROVIDE	fx.DotGraph <= go.uber.org/fx.(*App).dotGraph-fm()
	// [Fx] INVOKE		reflect.makeFuncStub()
}
