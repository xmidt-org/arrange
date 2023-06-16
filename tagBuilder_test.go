package arrange

import (
	"bytes"
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

func TestTagBuilder(t *testing.T) {
	suite.Run(t, new(TagBuilderSuite))
}
