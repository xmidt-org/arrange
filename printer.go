package arrange

import (
	"fmt"
	"io"
	"log"
	"os"

	"go.uber.org/fx"
)

// module is what code in this package passes to Prepend as its module parameter
const module = "Arrange"

// Prepend creates the standard format for information output that uber/fx uses.
// It returns a string of the form "[module] template".  This function can be used
// in conjunction with fx.Printer to standard informational output.
func Prepend(module, template string) string {
	return "[" + module + "] " + template
}

// PrinterFunc is a function type that implements fx.Printer.  This is useful
// for passing functions as printers, such as when using go.uber.org/zap with
// a SugaredLogger's methods.
type PrinterFunc func(string, ...interface{})

// Printf implements fx.Printer.  Note that this method does not append
// a newline to the output.
func (pf PrinterFunc) Printf(template string, args ...interface{}) {
	pf(template, args...)
}

// PrinterWriter creates an fx.Printer that sends all output to the specified
// Writer.  Each write has a newline appended.  Only one (1) write is performed
// for each call to Printf.
//
// Any error from Write() results in a panic.
func PrinterWriter(w io.Writer) fx.Printer {
	return PrinterFunc(func(template string, args ...interface{}) {
		_, err := fmt.Fprintf(w, template+"\n", args...)
		if err != nil {
			panic(err)
		}
	})
}

// defaultPrinter follows the same pattern as in the go.uber.org/fx/internal/fxlog package
var defaultPrinter fx.Printer = log.New(os.Stderr, "", log.LstdFlags)

// DefaultPrinter returns the fx.Printer that arrange uses when no printer
// is supplied.  This outputs to os.Stderr, in keeping with uber/fx's behavior.
func DefaultPrinter() fx.Printer {
	return defaultPrinter
}

// Logger is an analog to fx.Logger.  This version sets the logger with fx.Logger
// and, in addition, makes the printer available as a global, unnamed component.
// Code in this package and its subpackages will use this fx.Printer for informational
// output if supplied.
func Logger(p fx.Printer) fx.Option {
	return fx.Options(
		fx.Logger(p),

		// NOTE: Cannot use fx.Supply here, as that produces a component
		// of the most-derived type, e.g. PrinterFunc.  We want a component
		// of type fx.Printer regardless of the concrete type.
		fx.Provide(
			func() fx.Printer {
				return p
			},
		),
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

// LoggerWriter creates an fx.Printer with PrinterWriter and sets that as the
// fx.Logger and exposes it as an unnamed component.
func LoggerWriter(w io.Writer) fx.Option {
	return Logger(PrinterWriter(w))
}

// t is implemented by both *testing.T and *testing.B
type t interface {
	Name() string
	Logf(string, ...interface{})
}

// TestLogger uses Logger to establish a *testing.T or *testing.B as the
// sink for uber/fx and arrange logging
func TestLogger(t t) fx.Option {
	return LoggerFunc(
		func(template string, args ...interface{}) {
			t.Logf(t.Name()+" "+template, args...)
		},
	)
}
