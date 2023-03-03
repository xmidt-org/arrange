package arrange

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
)

type VisitDependenciesSuite struct {
	suite.Suite
}

func (suite *VisitDependenciesSuite) visitDependencies(deps []any, expecteds []DependencyVisitor) {
	counter := 0
	VisitDependencies(
		func(d Dependency) bool {
			suite.Require().GreaterOrEqual(len(expecteds), counter, "Too many calls to the visitor")
			v := expecteds[counter](d)
			counter++
			return v
		},
		deps...,
	)
}

func (suite *VisitDependenciesSuite) TestSimple() {
	var (
		buffer    = new(bytes.Buffer)
		expecteds = []DependencyVisitor{
			func(d Dependency) bool {
				suite.Empty(d.Name())
				suite.Empty(d.Group())
				suite.False(d.Optional())
				suite.Nil(d.Field)
				suite.Nil(d.Container)

				suite.Require().True(d.Value.IsValid())
				suite.Same(d.Value.Interface(), buffer)
				suite.True(d.Injected())
				suite.NotEmpty(d.String())

				return false // skip everything else
			},
		}
	)

	suite.Run("ReflectValues", func() {
		suite.visitDependencies(
			[]any{
				reflect.ValueOf(buffer), reflect.ValueOf("this should be skipped"),
			},
			expecteds,
		)
	})

	suite.Run("Raw", func() {
		suite.visitDependencies(
			[]any{
				buffer, "this should be skipped",
			},
			expecteds,
		)
	})
}

func (suite *VisitDependenciesSuite) TestIn() {
	type In struct {
		fx.In
		A *bytes.Buffer
		B *bytes.Buffer   `name:"named" optional:"true"`
		C []*bytes.Buffer `group:"buffers"`
	}

	var (
		in = In{
			A: bytes.NewBufferString("A"),
			C: []*bytes.Buffer{bytes.NewBufferString("C1"), bytes.NewBufferString("C2")},
		}

		expecteds = []DependencyVisitor{
			// A
			func(d Dependency) bool {
				suite.Empty(d.Name())
				suite.Empty(d.Group())
				suite.False(d.Optional())
				suite.NotNil(d.Field)
				suite.Equal(reflect.TypeOf(In{}), d.Container)
				suite.Require().True(d.Value.IsValid())
				suite.Same(in.A, d.Value.Interface())
				suite.True(d.Injected())
				suite.NotEmpty(d.String())
				return true
			},
			// B
			func(d Dependency) bool {
				suite.Equal("named", d.Name())
				suite.Empty(d.Group())
				suite.True(d.Optional())
				suite.NotNil(d.Field)
				suite.Equal(reflect.TypeOf(In{}), d.Container)
				suite.True(d.Value.IsValid())
				suite.False(d.Injected())
				suite.NotEmpty(d.String())
				return true
			},
			// C
			func(d Dependency) bool {
				suite.Empty(d.Name())
				suite.Equal("buffers", d.Group())
				suite.False(d.Optional())
				suite.NotNil(d.Field)
				suite.Equal(reflect.TypeOf(In{}), d.Container)
				suite.Require().True(d.Value.IsValid())
				suite.Equal(
					[]*bytes.Buffer{
						bytes.NewBufferString("C1"), bytes.NewBufferString("C2"),
					},
					d.Value.Interface(),
				)

				suite.True(d.Injected())
				suite.NotEmpty(d.String())
				return false
			},
		}
	)

	suite.Run("ReflectValues", func() {
		suite.visitDependencies(
			[]any{reflect.ValueOf(in), reflect.ValueOf("this should be skipped")},
			expecteds,
		)
	})

	suite.Run("Raw", func() {
		suite.visitDependencies(
			[]any{in, "this should be skipped"},
			expecteds,
		)
	})
}

func (suite *VisitDependenciesSuite) TestRecursion() {
	type Embedded struct {
		fx.In
		B *bytes.Buffer   `name:"named" optional:"true"`
		C []*bytes.Buffer `group:"buffers"`
	}

	type Recurse struct {
		A *bytes.Buffer
		Embedded
	}

	var (
		recurse = Recurse{
			A: bytes.NewBufferString("A"),
			Embedded: Embedded{
				C: []*bytes.Buffer{bytes.NewBufferString("C1"), bytes.NewBufferString("C2")},
			},
		}

		expecteds = []DependencyVisitor{
			// A
			func(d Dependency) bool {
				suite.Empty(d.Name())
				suite.Empty(d.Group())
				suite.False(d.Optional())
				suite.NotNil(d.Field)
				suite.Equal(reflect.TypeOf(Recurse{}), d.Container)
				suite.Require().True(d.Value.IsValid())
				suite.Same(recurse.A, d.Value.Interface())
				suite.True(d.Injected())
				suite.NotEmpty(d.String())
				return true
			},
			// B
			func(d Dependency) bool {
				suite.Equal("named", d.Name())
				suite.Empty(d.Group())
				suite.True(d.Optional())
				suite.NotNil(d.Field)
				suite.Equal(reflect.TypeOf(Embedded{}), d.Container)
				suite.Same(recurse.Embedded.B, d.Value.Interface())
				suite.True(d.Value.IsValid())
				suite.False(d.Injected())
				suite.NotEmpty(d.String())
				return true
			},
			// C
			func(d Dependency) bool {
				suite.Empty(d.Name())
				suite.Equal("buffers", d.Group())
				suite.False(d.Optional())
				suite.NotNil(d.Field)
				suite.Equal(reflect.TypeOf(Embedded{}), d.Container)
				suite.Require().True(d.Value.IsValid())
				suite.Equal(
					[]*bytes.Buffer{bytes.NewBufferString("C1"), bytes.NewBufferString("C2")},
					d.Value.Interface(),
				)

				suite.True(d.Injected())
				suite.NotEmpty(d.String())
				return false
			},
		}
	)

	suite.Run("ReflectValues", func() {
		suite.visitDependencies(
			[]any{reflect.ValueOf(recurse), reflect.ValueOf("this should be skipped")},
			expecteds,
		)
	})

	suite.Run("Raw", func() {
		suite.visitDependencies(
			[]any{recurse, "this should be skipped"},
			expecteds,
		)
	})
}

func TestVisitDependencies(t *testing.T) {
	suite.Run(t, new(VisitDependenciesSuite))
}
