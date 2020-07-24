package arrange

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

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

type alwaysError struct{}

func (ae alwaysError) Write([]byte) (int, error) {
	return 0, errors.New("expected")
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
