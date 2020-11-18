package arrange

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

type InjectTestSuite struct {
	suite.Suite
}

func (suite *InjectTestSuite) TestEmpty() {
	suite.Empty(Inject{}.Types())

	ft := Inject{}.FuncOf()
	suite.Require().NotNil(ft)
	suite.Require().Equal(reflect.Func, ft.Kind())
	suite.Equal(0, ft.NumIn())
	suite.Equal(0, ft.NumOut())
}

func (suite *InjectTestSuite) TestFuncOf() {
	var (
		s = Struct{}.In().Append(Field{
			Name: "component",
			Type: (*bytes.Buffer)(nil),
		})

		ij = Inject{}.Append(
			(*bytes.Buffer)(nil),
			s.Of(),
		)

		ft = ij.FuncOf(reflect.TypeOf(int(0)), ErrorType())
	)

	suite.Require().NotNil(ft)
	suite.Require().Equal(reflect.Func, ft.Kind())

	suite.Equal(2, ft.NumIn())
	suite.Equal(reflect.TypeOf((*bytes.Buffer)(nil)), ft.In(0))
	suite.Equal(s.Of(), ft.In(1))

	suite.Equal(2, ft.NumOut())
	suite.Equal(reflect.TypeOf(int(0)), ft.Out(0))
	suite.Equal(ErrorType(), ft.Out(1))
}

func (suite *InjectTestSuite) TestMakeFunc() {
	var (
		buffer = new(bytes.Buffer)

		ij = Inject{
			(*bytes.Buffer)(nil),
			reflect.ValueOf(int(0)),
		}

		fv = ij.MakeFunc(
			func(inputs []reflect.Value) (string, error) {
				suite.Require().Len(inputs, 2)
				suite.Equal(buffer, inputs[0].Interface())
				suite.Equal(123, inputs[1].Interface())
				return "test", nil
			},
		)
	)

	outputs := fv.Call(
		[]reflect.Value{
			reflect.ValueOf(buffer),
			reflect.ValueOf(123),
		},
	)

	suite.Require().Len(outputs, 2)
	suite.Equal("test", outputs[0].Interface())
	suite.True(outputs[1].IsNil())
}

func (suite *InjectTestSuite) TestExtend() {
	ij := Inject{}.Extend(
		Inject{
			(*bytes.Buffer)(nil),
			123,
		},
	)

	suite.Require().Len(ij, 2)
	suite.Equal(
		ij,
		Inject{
			(*bytes.Buffer)(nil),
			123,
		},
	)
}

func TestInject(t *testing.T) {
	suite.Run(t, new(InjectTestSuite))
}
