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

// Logger is an analog to fx.Logger.  This version sets the logger with fx.Logger
// and, in addition, makes the printer available as a global, unnamed component.
// Code in this package and its subpackages will use this fx.Printer for informational
// output if supplied.
func Logger(p fx.Printer) fx.Option {
	return fx.Options(
		fx.Logger(p),
		fx.Supply(p),
	)
}

// LoggerFunc is like Logger, but provides syntactic sugar around supplying
// closure that can do printing.
//
// A great use of this is with go.uber.org/zap:
//
//   l := zap.NewDevelopment() // or any zap logger
//   fx.New(
//     // DI container logging will go to the above logger at the INFO level
//     arrange.LoggerFunc(l.Sugar().Infof),
//
//     // the zap logger will now be used for both uber/fx and arrange messages
//   )
func LoggerFunc(pf PrinterFunc) fx.Option {
	return Logger(pf)
}

// LoggerWriter uses Logger to supply an fx.Printer that writes to an io.Writer.
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
		func(template string, args ...interface{}) {
			_, err := fmt.Fprintf(w, template, args...)
			if err == nil {
				_, err = w.Write(newline)
			}

			if err != nil {
				panic(err)
			}
		},
	)
}

type t interface {
	Logf(string, ...interface{})
}

// TestLogger uses Logger to establish a *testing.T or *testing.B as the
// sink for uber/fx and arrange logging
func TestLogger(t t) fx.Option {
	return LoggerFunc(
		func(template string, args ...interface{}) {
			t.Logf(template, args...)
		},
	)
}
