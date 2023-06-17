package arrange

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type TagBuilderSuite struct {
	suite.Suite
}

func (suite *TagBuilderSuite) TestParamTags() {
	type parameters struct {
		fx.Out

		Named  string   `name:"name"`
		Values []string `group:"values"`
	}

	var buffer *bytes.Buffer
	app := fxtest.New(
		suite.T(),
		fx.Provide(
			func() parameters {
				return parameters{} // doesn't matter what the values are
			},
			func() int { return 123 },
			fx.Annotate(
				func(
					name string, optional string, values []string, optionalUnnamed string, skipped int,
				) *bytes.Buffer {
					return new(bytes.Buffer) // dummy component
				},
				Tags().
					Name("name").
					OptionalName("optional").
					Group("values").
					Optional().
					Skip().
					ParamTags(),
			),
		),
		fx.Populate(&buffer), // force the constructor to run
	)

	app.RequireStart()
	app.RequireStop()
	suite.NotNil(buffer)
}

func (suite *TagBuilderSuite) TestResultTags() {
	type populate struct {
		fx.In
		Named   *bytes.Buffer   `name:"named"`
		Buffers []*bytes.Buffer `group:"buffers"`
	}

	var p populate

	app := fxtest.New(
		suite.T(),
		fx.Provide(
			fx.Annotate(
				func() *bytes.Buffer { return new(bytes.Buffer) },
				Tags().Name("named").ResultTags(),
			),
			fx.Annotate(
				func() *bytes.Buffer { return new(bytes.Buffer) },
				Tags().Group("buffers").ResultTags(),
			),
			fx.Annotate(
				func() *bytes.Buffer { return new(bytes.Buffer) },
				Tags().Group("buffers").ResultTags(),
			),
		),
		fx.Invoke(
			func(in populate) {
				p = in
			},
		),
	)

	app.RequireStart()
	app.RequireStop()
	suite.NotNil(p.Named)
	suite.Len(p.Buffers, 2)
}

func (suite *TagBuilderSuite) assertStructTag(st reflect.StructTag, expectedName, expectedGroup string, optional bool) {
	v, ok := st.Lookup("optional")
	if optional {
		suite.Equal("true", v)
	} else {
		suite.False(ok)
	}

	v, ok = st.Lookup("name")
	if len(expectedName) > 0 {
		suite.Equal(expectedName, v)
	} else {
		suite.False(ok)
	}

	v, ok = st.Lookup("group")
	if len(expectedGroup) > 0 {
		suite.Equal(expectedGroup, v)
	} else {
		suite.False(ok)
	}
}

func (suite *TagBuilderSuite) TestStructTags() {
	tags := Tags().
		Optional().
		Skip().
		Group("group").
		Name("name").
		OptionalName("optional").
		StructTags()

	suite.Require().Len(tags, 5)
	suite.assertStructTag(tags[0], "", "", true)
	suite.Empty(string(tags[1]))
	suite.assertStructTag(tags[2], "", "group", false)
	suite.assertStructTag(tags[3], "name", "", false)
	suite.assertStructTag(tags[4], "optional", "", true)
}

func TestTagBuilder(t *testing.T) {
	suite.Run(t, new(TagBuilderSuite))
}
