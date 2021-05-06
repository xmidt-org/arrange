package arrange

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestNotAFunctionError(t *testing.T) {
	err := &NotAFunctionError{Type: reflect.TypeOf(123)}
	assert.NotEmpty(t, err.Error())
}

type BindTestSuite struct {
	suite.Suite
}

func (suite *BindTestSuite) TestEmpty() {
	app := fxtest.New(
		suite.T(),
		Bind{}.With(),
	)

	suite.NoError(app.Err())
	app.RequireStart()
	app.RequireStop()
}

func (suite *BindTestSuite) TestNotAFunction() {
	called := 0
	app := fx.New(
		DiscardLogger(),
		Bind{
			func() {
				// this will be fine
				called++
			},
			"this is not a function",
		}.With(),
	)

	suite.Error(app.Err())
	suite.Equal(0, called) // should have shortcircuited
}

func (suite *BindTestSuite) TestWithNoArgs() {
	var (
		called = 0

		supplied = new(bytes.Buffer)
		created  = new(strings.Reader)

		expectedValue = 123
		actualValue   int

		app = fxtest.New(
			suite.T(),
			fx.Supply(supplied),
			Bind{
				func() {
					called++
				},
				func() error {
					called++
					return nil
				},
				func(actual *bytes.Buffer) {
					called++
					suite.Same(supplied, actual)
				},
				func(actual *bytes.Buffer) error {
					called++
					suite.Same(supplied, actual)
					return nil
				},
				func() (*strings.Reader, error) {
					called++
					return created, nil
				},
				func(arg0 *strings.Reader, arg1 *bytes.Buffer) {
					called++
					suite.Same(created, arg0)
					suite.Same(supplied, arg1)
				},
				func(actual *strings.Reader) int {
					called++
					return expectedValue
				},
			}.With(),
			fx.Populate(&actualValue),
		)
	)

	suite.Require().NoError(app.Err())
	suite.Equal(expectedValue, actualValue)

	app.RequireStart()
	app.RequireStop()

	suite.Equal(7, called)
}

func (suite *BindTestSuite) TestWithArgs() {
	var (
		called = 0

		name   = "this is a name"
		buffer = new(bytes.Buffer)

		created       = new(strings.Reader)
		expectedValue = 948573
		actualValue   int

		app = fxtest.New(
			suite.T(),
			Bind{
				func() {
					called++
				},
				func() error {
					called++
					return nil
				},
				func(actual *bytes.Buffer) {
					called++
					suite.Same(buffer, actual)
				},
				func(actual *bytes.Buffer) error {
					called++
					suite.Same(buffer, actual)
					return nil
				},
				func(actual string) {
					called++
					suite.Equal(name, actual)
				},
				func(actual string) error {
					called++
					suite.Equal(name, actual)
					return nil
				},
				func(arg0 string, arg1 *bytes.Buffer) (*strings.Reader, error) {
					called++
					suite.Equal(name, arg0)
					suite.Same(buffer, arg1)

					return created, nil
				},
				func(actual *strings.Reader, arg0 string, arg1 *bytes.Buffer) {
					called++
					suite.Equal(name, arg0)
					suite.Same(buffer, arg1)

					suite.Same(created, actual)
				},
				func() int {
					called++
					return expectedValue
				},
			}.With(name, buffer),
			fx.Populate(&actualValue),
		)
	)

	suite.Require().NoError(app.Err())
	suite.Equal(expectedValue, actualValue)

	app.RequireStart()
	app.RequireStop()

	suite.Equal(9, called)
}

func TestBind(t *testing.T) {
	suite.Run(t, new(BindTestSuite))
}
