package arrange

import (
	"fmt"
	"io"

	"go.uber.org/fx"
)

var newline = []byte{'\n'}

// PrinterFunc is a function type that implements fx.Printer.  This is useful
// for passing functions as printers, such as when using go.uber.org/zap with
// a SugaredLogger's methods.
type PrinterFunc func(string, ...interface{})

// Printf implements fx.Printer
func (pf PrinterFunc) Printf(template string, args ...interface{}) {
	pf(template, args...)
}

// LoggerWriter supplies an fx.Printer that writes to an arbitrary io.Writer.
// Any write error results in a panic.  Every write has a newline appended to it.
//
// This is a convenient function to dump all DI container logging to os.Stdout
// or os.Stderr:
//
//   fx.New(
//     // NOTE: the default setup for go.uber.org/fx sends output to os.Stderr
//     arrange.LoggerWriter(os.Stdout),
//
//     // carry on ...
//   )
func LoggerWriter(w io.Writer) fx.Option {
	return LoggerFunc(
		PrinterFunc(func(template string, args ...interface{}) {
			_, err := fmt.Fprintf(w, template, args...)
			if err == nil {
				_, err = w.Write(newline)
			}

			if err != nil {
				panic(err)
			}
		}),
	)
}

// LoggerFunc is an analog of fx.Logger, but for use with functions that
// can do printing.  A great use of this is with go.uber.org/zap:
//
//   l := zap.NewDevelopment() // or any zap logger
//   fx.New(
//     // DI container logging will go to the above logger at the INFO level
//     arrange.LoggerFunc(l.Sugar().Infof),
//
//     // carry on ...
//   )
//
// You can do the same thing with fx.Logger, but a cast is needed.  This
// function simply provides a cleaner, less noisy way of specifying a PrinterFunc.
func LoggerFunc(pf PrinterFunc) fx.Option {
	return fx.Logger(pf)
}
