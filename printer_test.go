package arrange

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestPrinterFunc(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		output  bytes.Buffer

		pf = func(template string, args ...interface{}) {
			_, err := fmt.Fprintf(&output, template, args...)
			require.NoError(err)
		}
	)

	PrinterFunc(pf).Printf("test %d", 123)
	assert.Equal("test 123", output.String())
}

func testNewPrinterWriterBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		output  bytes.Buffer

		pw = NewPrinterWriter(&output)
	)

	require.NotNil(pw)
	pw.Printf("test: %d", 123)
	assert.Equal("test: 123\n", output.String())
}

func testNewPrinterWriterError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		pw = NewPrinterWriter(badWriter{})
	)

	require.NotNil(pw)
	assert.Panics(func() {
		pw.Printf("test: %d", 123)
	})
}

func TestNewPrinterWriter(t *testing.T) {
	t.Run("Basic", testNewPrinterWriterBasic)
	t.Run("Error", testNewPrinterWriterError)
}

func testNewModulePrinterBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		output  bytes.Buffer
		printer fx.Printer
	)

	app := fx.New(
		LoggerWriter(&output),
		fx.Provide(
			func() int { return 1 },
		),
		fx.Populate(&printer),
	)

	require.NoError(app.Err())
	mp := NewModulePrinter("TEST", printer)
	require.NotNil(mp)

	mp.Printf("test: %d", 123)
	require.NotEmpty(output.String())
	assert.Contains(output.String(), "[TEST]")
	assert.Contains(output.String(), "test: 123")
}

func testNewModulePrinterDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		output bytes.Buffer
	)

	old := defaultPrinter
	defer func() {
		defaultPrinter = old
	}()

	defaultPrinter = PrinterFunc(func(template string, args ...interface{}) {
		_, err := fmt.Fprintf(&output, template, args...)
		require.NoError(err)
	})

	mp := NewModulePrinter("TEST", nil)
	require.NotNil(mp)

	mp.Printf("test: %d", 123)
	require.NotEmpty(output.String())
	assert.Contains(output.String(), "[TEST]")
	assert.Contains(output.String(), "test: 123")
}

func TestNewModulePrinter(t *testing.T) {
	t.Run("Basic", testNewModulePrinterBasic)
	t.Run("Default", testNewModulePrinterDefault)
}

func TestDefaultPrinter(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(defaultPrinter, DefaultPrinter())
}

func TestLoggerWriter(t *testing.T) {
	var (
		assert = assert.New(t)
		output bytes.Buffer
	)

	fxtest.New(
		t,
		LoggerWriter(&output),
	)

	assert.NotEmpty(output.String())
}

func TestLoggerFunc(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		output  bytes.Buffer
		printer fx.Printer
	)

	fxtest.New(
		t,
		LoggerFunc(
			func(template string, args ...interface{}) {
				_, err := fmt.Fprintf(&output, template, args...)
				require.NoError(err)
			},
		),
		fx.Populate(&printer),
	)

	assert.NotEmpty(output.String())
}

func TestTestLogger(t *testing.T) {
	var printer fx.Printer
	fxtest.New(
		t,
		TestLogger(t),
		fx.Populate(&printer),
	)
}
