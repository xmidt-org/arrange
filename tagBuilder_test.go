/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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

		Named          string   `name:"name"`
		Values         []string `group:"values"`
		PrefixedNamed  string   `name:"prefix.name"`
		PrefixedValues []string `group:"prefix.values"`
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
					name, prefixedName string, optional string, values, prefixedValues []string, optionalUnnamed string, skipped int,
				) *bytes.Buffer {
					return new(bytes.Buffer) // dummy component
				},
				Tags().
					Name("name").
					Push("prefix").Name("name").Pop().
					OptionalName("optional").
					Group("values").
					Push("prefix").Group("values").Pop().
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
		Named           *bytes.Buffer   `name:"named"`
		Buffers         []*bytes.Buffer `group:"buffers"`
		PrefixedNamed   *bytes.Buffer   `name:"prefix.named"`
		PrefixedBuffers []*bytes.Buffer `group:"prefix.buffers"`
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
			fx.Annotate(
				func() *bytes.Buffer { return new(bytes.Buffer) },
				Tags().Push("prefix").Name("named").ResultTags(),
			),
			fx.Annotate(
				func() *bytes.Buffer { return new(bytes.Buffer) },
				Tags().Push("prefix").Group("buffers").ResultTags(),
			),
			fx.Annotate(
				func() *bytes.Buffer { return new(bytes.Buffer) },
				Tags().Push("prefix").Group("buffers").ResultTags(),
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
	suite.NotNil(p.PrefixedNamed)
	suite.Len(p.Buffers, 2)
	suite.Len(p.PrefixedBuffers, 2)
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
		Push("prefix1").Name("name").
		Push("prefix2").Name("name").
		Pop().Pop().Name("notprefixed").
		StructTags()

	suite.Require().Len(tags, 8)
	suite.assertStructTag(tags[0], "", "", true)
	suite.Empty(string(tags[1]))
	suite.assertStructTag(tags[2], "", "group", false)
	suite.assertStructTag(tags[3], "name", "", false)
	suite.assertStructTag(tags[4], "optional", "", true)
	suite.assertStructTag(tags[5], "prefix1.name", "", false)
	suite.assertStructTag(tags[6], "prefix2.name", "", false)
	suite.assertStructTag(tags[7], "notprefixed", "", false)
}

func TestTagBuilder(t *testing.T) {
	suite.Run(t, new(TagBuilderSuite))
}
