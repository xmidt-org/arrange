package arrange

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
)

type StructTestSuite struct {
	suite.Suite
}

func (suite *StructTestSuite) TestEmpty() {
	st := Struct{}.Of()
	suite.Require().NotNil(st)
	suite.Require().Equal(reflect.Struct, st.Kind())
	suite.Zero(st.NumField())
}

func (suite *StructTestSuite) TestSeveral() {
	st := Struct{}.
		Append(
			Field{
				Type: (*bytes.Buffer)(nil),
			},
			Field{
				Type:     (*bytes.Buffer)(nil),
				Optional: true,
			},
			Field{
				Name: "component1",
				Type: (*bytes.Buffer)(nil),
			},
			Field{
				Name:     "component2",
				Optional: true,
				Type:     reflect.TypeOf((*bytes.Buffer)(nil)),
			},
			Field{
				Group: "buffers",
				Type:  reflect.ValueOf((*bytes.Buffer)(nil)),
			},
		).In().Of()

	suite.Require().NotNil(st)
	suite.Require().Equal(reflect.Struct, st.Kind())
	suite.Equal(6, st.NumField())

	{
		f := st.Field(0)
		suite.Equal(reflect.TypeOf((*bytes.Buffer)(nil)), f.Type)
		suite.Empty(f.PkgPath)
		suite.False(f.Anonymous)
		suite.Empty(f.Tag)
	}

	{
		f := st.Field(1)
		suite.Equal(reflect.TypeOf((*bytes.Buffer)(nil)), f.Type)
		suite.Empty(f.PkgPath)
		suite.False(f.Anonymous)
		suite.Empty(f.Tag.Get("name"))
		suite.Empty(f.Tag.Get("group"))
		suite.Equal("true", f.Tag.Get("optional"))
	}

	{
		f := st.Field(2)
		suite.Equal(reflect.TypeOf((*bytes.Buffer)(nil)), f.Type)
		suite.Empty(f.PkgPath)
		suite.False(f.Anonymous)
		suite.Equal("component1", f.Tag.Get("name"))
		suite.Empty(f.Tag.Get("group"))
		suite.Empty(f.Tag.Get("optional"))
	}

	{
		f := st.Field(3)
		suite.Equal(reflect.TypeOf((*bytes.Buffer)(nil)), f.Type)
		suite.Empty(f.PkgPath)
		suite.False(f.Anonymous)
		suite.Equal("component2", f.Tag.Get("name"))
		suite.Empty(f.Tag.Get("group"))
		suite.Equal("true", f.Tag.Get("optional"))
	}

	{
		f := st.Field(4)
		suite.Require().Equal(reflect.Slice, f.Type.Kind())
		suite.Equal(reflect.TypeOf((*bytes.Buffer)(nil)), f.Type.Elem())
		suite.Empty(f.PkgPath)
		suite.False(f.Anonymous)
		suite.Empty(f.Tag.Get("name"))
		suite.Equal("buffers", f.Tag.Get("group"))
		suite.Empty(f.Tag.Get("optional"))
	}

	{
		f := st.Field(5)
		suite.Equal(Type[fx.In](), f.Type)
		suite.Empty(f.PkgPath)
		suite.True(f.Anonymous)
	}
}

func (suite *StructTestSuite) TestExtend() {
	s1 := Struct{}.In().
		Append(Field{
			Name: "component",
			Type: (*bytes.Buffer)(nil),
		})

	s2 := Struct{}.Append(
		Field{
			Name: "extended1",
			Type: (*bytes.Buffer)(nil),
		},
	)

	s3 := s1.Extend(s2)
	suite.Len(s3, 3)
	suite.Equal(s1[0], s3[0])
	suite.Equal(s1[1], s3[1])
	suite.Equal(s2[0], s3[2])
}

func (suite *StructTestSuite) TestClone() {
	s1 := Struct{}.In().
		Append(Field{
			Name: "component",
			Type: (*bytes.Buffer)(nil),
		})

	s2 := s1.Clone()
	suite.Equal(s1, s2)
}

func TestStruct(t *testing.T) {
	suite.Run(t, new(StructTestSuite))
}
