package arrange

import (
	"fmt"
	"io"
	"log"
	"os"

	"go.uber.org/fx"
)

// Module is the fx.Printer module (NOT the golang module) for all output in this package
const Module = "Arrange"

// PrinterFunc is a function type that implements fx.Printer.  This is useful
// for passing functions as printers, such as when using go.uber.org/zap with
// a SugaredLogger's methods.
type PrinterFunc func(string, ...interface{})

// Printf implements fx.Printer.  Note that this method does not append
// a newline to the output.
func (pf PrinterFunc) Printf(template string, args ...interface{}) {
	pf(template, args...)
}

// NewPrinterWriter creates an fx.Printer that writes to the given writer.  Each call
// to Printf results in exactly one call to Write.  A single newline is appended to each
// line of output.  Any error from the io.Writer results in a panic.
func NewPrinterWriter(w io.Writer) fx.Printer {
	return PrinterFunc(func(template string, args ...interface{}) {
		_, err := fmt.Fprintf(w, template+"\n", args...)
		if err != nil {
			panic(err)
		}
	})
}

// NewModulePrinter decorates a given printer and prefixes "[module] " to
// each line of log output.  This adheres to the uber/fx de facto standard.
//
// If printer is nil, DefaultPrinter() will be used instead.
//
// This function's returned fx.Printer should not be used as a component for
// uber/fx.  This function is intended for packages which want to leverage
// an fx.Printer for their own informational output.
func NewModulePrinter(module string, printer fx.Printer) fx.Printer {
	if printer == nil {
		printer = DefaultPrinter()
	}

	prefix := "[" + module + "] "
	return PrinterFunc(func(template string, args ...interface{}) {
		printer.Printf(prefix+template, args...)
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

// DiscardLogger configures uber/fx to use an fx.Printer that throws away all input.
// It also makes that same fx.Printer available as a global component for dependency injection.
func DiscardLogger() fx.Option {
	return LoggerFunc(func(string, ...interface{}) {})
}

// LoggerWriter uses NewPrinterWriter to create an fx.Printer that writes to the
// given io.Writer.  The fx.Printer is made available as a component, and fx.Logger is
// used to set that printer as uber/fx's sink for informational output.
func LoggerWriter(w io.Writer) fx.Option {
	return Logger(
		NewPrinterWriter(w),
	)
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
