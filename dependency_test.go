package arrange

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func testVisitDependenciesSimple(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		expected = new(bytes.Buffer)

		called bool
	)

	VisitDependencies(
		func(d Dependency) bool {
			if called {
				assert.Fail("the visitor should not have been called after returning false")
				return false
			}

			t.Log("visited", d)
			assert.Empty(d.Name())
			assert.Empty(d.Group())
			assert.False(d.Optional())
			assert.Nil(d.Field)
			assert.Nil(d.Container)

			require.True(d.Value.IsValid())
			assert.Equal(d.Value.Interface(), expected)
			assert.True(d.Injected())
			assert.NotEmpty(d.String())

			called = true
			return false // skip everything else
		},
		reflect.ValueOf(expected), reflect.ValueOf(new(http.Request)),
	)
}

func testVisitDependenciesIn(t *testing.T) {
	type In struct {
		fx.In
		A *bytes.Buffer
		B *bytes.Buffer   `name:"named" optional:"true"`
		C []*bytes.Buffer `group:"buffers"`
	}

	type Skipped struct {
		fx.In
		B *bytes.Buffer
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)

		in = In{
			A: bytes.NewBufferString("A"),
			C: []*bytes.Buffer{bytes.NewBufferString("C1"), bytes.NewBufferString("C2")},
		}

		expecteds = []DependencyVisitor{
			// A
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Empty(d.Name())
				assert.Empty(d.Group())
				assert.False(d.Optional())
				assert.NotNil(d.Field)
				assert.Equal(reflect.TypeOf(In{}), d.Container)
				require.True(d.Value.IsValid())
				assert.Equal(bytes.NewBufferString("A"), d.Value.Interface())
				assert.True(d.Injected())
				assert.NotEmpty(d.String())
				return true
			},
			// B
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Equal("named", d.Name())
				assert.Empty(d.Group())
				assert.True(d.Optional())
				assert.NotNil(d.Field)
				assert.Equal(reflect.TypeOf(In{}), d.Container)
				assert.True(d.Value.IsValid())
				assert.False(d.Injected())
				assert.NotEmpty(d.String())
				return true
			},
			// C
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Empty(d.Name())
				assert.Equal("buffers", d.Group())
				assert.False(d.Optional())
				assert.NotNil(d.Field)
				assert.Equal(reflect.TypeOf(In{}), d.Container)
				require.True(d.Value.IsValid())
				assert.Equal(
					[]*bytes.Buffer{
						bytes.NewBufferString("C1"), bytes.NewBufferString("C2"),
					},
					d.Value.Interface(),
				)

				assert.True(d.Injected())
				assert.NotEmpty(d.String())
				return false
			},
		}

		counter int
	)

	VisitDependencies(
		func(d Dependency) bool {
			if counter >= len(expecteds) {
				assert.Fail("Too many calls to the visitor")
				return false
			}

			v := expecteds[counter](d)
			counter++
			return v
		},
		reflect.ValueOf(in), reflect.ValueOf(Skipped{}),
	)
}

func testVisitDependenciesRecursion(t *testing.T) {
	type Embedded struct {
		fx.In
		B *bytes.Buffer   `name:"named" optional:"true"`
		C []*bytes.Buffer `group:"buffers"`
	}

	type Recurse struct {
		A *bytes.Buffer
		Embedded
	}

	type Skipped struct {
		fx.In
		B *bytes.Buffer
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)

		recurse = Recurse{
			A: bytes.NewBufferString("A"),
			Embedded: Embedded{
				C: []*bytes.Buffer{bytes.NewBufferString("C1"), bytes.NewBufferString("C2")},
			},
		}

		expecteds = []DependencyVisitor{
			// A
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Empty(d.Name())
				assert.Empty(d.Group())
				assert.False(d.Optional())
				assert.NotNil(d.Field)
				assert.Equal(reflect.TypeOf(Recurse{}), d.Container)
				require.True(d.Value.IsValid())
				assert.Equal(bytes.NewBufferString("A"), d.Value.Interface())
				assert.True(d.Injected())
				assert.NotEmpty(d.String())
				return true
			},
			// B
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Equal("named", d.Name())
				assert.Empty(d.Group())
				assert.True(d.Optional())
				assert.NotNil(d.Field)
				assert.Equal(reflect.TypeOf(Embedded{}), d.Container)
				assert.True(d.Value.IsValid())
				assert.False(d.Injected())
				assert.NotEmpty(d.String())
				return true
			},
			// C
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Empty(d.Name())
				assert.Equal("buffers", d.Group())
				assert.False(d.Optional())
				assert.NotNil(d.Field)
				assert.Equal(reflect.TypeOf(Embedded{}), d.Container)
				require.True(d.Value.IsValid())
				assert.Equal(
					[]*bytes.Buffer{
						bytes.NewBufferString("C1"), bytes.NewBufferString("C2"),
					},
					d.Value.Interface(),
				)

				assert.True(d.Injected())
				assert.NotEmpty(d.String())
				return false
			},
		}

		counter int
	)

	VisitDependencies(
		func(d Dependency) bool {
			if counter >= len(expecteds) {
				assert.Fail("Too many calls to the visitor")
				return false
			}

			v := expecteds[counter](d)
			counter++
			return v
		},
		reflect.ValueOf(recurse), reflect.ValueOf(Skipped{}),
	)
}

func TestVisitDependencies(t *testing.T) {
	t.Run("Simple", testVisitDependenciesSimple)
	t.Run("In", testVisitDependenciesIn)
	t.Run("Recursion", testVisitDependenciesRecursion)
}
