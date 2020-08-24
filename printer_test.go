package arrange

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// alwaysError is an io.Writer that always returns an error
type alwaysError struct{}

func (ae alwaysError) Write([]byte) (int, error) {
	return 0, errors.New("expected io.Writer error")
}

func TestPrepend(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("[Test] MODULE", Prepend("Test", "MODULE"))
}

func TestPrinterFunc(t *testing.T) {
	var (
		assert = assert.New(t)
		output bytes.Buffer

		pf = func(template string, args ...interface{}) {
			fmt.Fprintf(&output, template, args...)
		}
	)

	PrinterFunc(pf).Printf("test %d", 123)
	assert.Equal("test 123", output.String())
}

func TestDefaultPrinter(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(defaultPrinter, DefaultPrinter())
}

func TestGetPrinter(t *testing.T) {
	var (
		assert   = assert.New(t)
		expected = log.New(ioutil.Discard, "", 0)
	)

	assert.Equal(DefaultPrinter(), GetPrinter(nil))
	assert.Equal(expected, GetPrinter(expected))
}

func testPrinterWriterSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		output  bytes.Buffer

		pw = PrinterWriter(&output)
	)

	require.NotNil(pw)
	pw.Printf("test %d", 123)
	assert.Equal("test 123\n", output.String())
}

func testPrinterWriterError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		pw = PrinterWriter(alwaysError{})
	)

	require.NotNil(pw)
	assert.Panics(func() {
		pw.Printf("test %d", 123)
	})
}

func TestPrinterWriter(t *testing.T) {
	t.Run("Success", testPrinterWriterSuccess)
	t.Run("Error", testPrinterWriterError)
}

func testLoggerWriterSuccess(t *testing.T) {
	var (
		assert = assert.New(t)
		output bytes.Buffer

		dummy int
	)

	fxtest.New(
		t,
		LoggerWriter(&output),

		// this is just to force some logging.  it doesn't matter what
		// the component is
		fx.Supply(123),
		fx.Populate(&dummy),
	)

	assert.Greater(output.Len(), 0)
}

func testLoggerWriterError(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		var dummy int
		fxtest.New(
			t,
			LoggerWriter(alwaysError{}),

			// this is just to force some logging.  it doesn't matter what
			// the component is
			fx.Supply(123),
			fx.Populate(&dummy),
		)
	})
}

func TestLoggerWriter(t *testing.T) {
	t.Run("Success", testLoggerWriterSuccess)
	t.Run("Error", testLoggerWriterError)
}

func TestLoggerFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		printerCalled bool
		printerFunc   = func(template string, args ...interface{}) {
			printerCalled = true
		}

		dummy int
	)

	fxtest.New(
		t,
		LoggerFunc(printerFunc),

		// this is just to force some logging.  it doesn't matter what
		// the component is
		fx.Supply(123),
		fx.Populate(&dummy),
	)

	assert.True(printerCalled)
}

func TestTestLogger(t *testing.T) {
	var (
		assert = assert.New(t)
		dummy  string
	)

	fxtest.New(
		t,
		TestLogger(t),
		fx.Supply("test"),
		fx.Populate(&dummy),
	)

	assert.Equal("test", dummy)
}
