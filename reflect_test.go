package arrange

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

type NewTargetTester struct {
	Name string
	Age  int
}

func testNewTargetValue(t *testing.T) {
	testData := []NewTargetTester{
		{},
		{
			Name: "testy mctest",
			Age:  45,
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert = assert.New(t)
				target = NewTarget(record)
			)

			assert.Equal(
				target.Component.Interface(),
				record,
			)

			assert.Equal(
				target.Component.Type(),
				reflect.TypeOf(NewTargetTester{}),
			)

			assert.Equal(
				record,
				*target.UnmarshalTo.Interface().(*NewTargetTester),
			)
		})
	}
}

func testNewTargetPointer(t *testing.T) {
	testData := []NewTargetTester{
		{},
		{
			Name: "testy mctest",
			Age:  45,
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert = assert.New(t)
				target = NewTarget(&record)
			)

			assert.Equal(
				target.Component.Interface(),
				&record,
			)

			assert.Equal(
				target.Component.Type(),
				reflect.TypeOf((*NewTargetTester)(nil)),
			)

			assert.Equal(
				&record,
				target.UnmarshalTo.Interface().(*NewTargetTester),
			)
		})
	}
}

func testNewTargetNil(t *testing.T) {
	var (
		assert = assert.New(t)
		target = NewTarget((*NewTargetTester)(nil))
	)

	assert.Equal(
		target.Component.Interface(),
		&NewTargetTester{},
	)

	assert.Equal(
		target.Component.Type(),
		reflect.TypeOf((*NewTargetTester)(nil)),
	)

	assert.Equal(
		&NewTargetTester{},
		target.UnmarshalTo.Interface().(*NewTargetTester),
	)
}

func TestNewTarget(t *testing.T) {
	t.Run("Value", testNewTargetValue)
	t.Run("Pointer", testNewTargetPointer)
	t.Run("Nil", testNewTargetNil)
}

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
			assert.Empty(d.Name)
			assert.Empty(d.Group)
			assert.False(d.Optional)
			assert.Empty(d.Tag)
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
				assert.Empty(d.Name)
				assert.Empty(d.Group)
				assert.False(d.Optional)
				assert.Empty(d.Tag)
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
				assert.Equal("named", d.Name)
				assert.Empty(d.Group)
				assert.True(d.Optional)
				assert.Equal(
					reflect.StructTag(`name:"named" optional:"true"`),
					d.Tag,
				)

				assert.Equal(reflect.TypeOf(In{}), d.Container)
				assert.True(d.Value.IsValid())
				assert.False(d.Injected())
				assert.NotEmpty(d.String())
				return true
			},
			// C
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Empty(d.Name)
				assert.Equal("buffers", d.Group)
				assert.False(d.Optional)
				assert.Equal(
					reflect.StructTag(`group:"buffers"`),
					d.Tag,
				)

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
				assert.Empty(d.Name)
				assert.Empty(d.Group)
				assert.False(d.Optional)
				assert.Empty(d.Tag)
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
				assert.Equal("named", d.Name)
				assert.Empty(d.Group)
				assert.True(d.Optional)
				assert.Equal(
					reflect.StructTag(`name:"named" optional:"true"`),
					d.Tag,
				)

				assert.Equal(reflect.TypeOf(Embedded{}), d.Container)
				assert.True(d.Value.IsValid())
				assert.False(d.Injected())
				assert.NotEmpty(d.String())
				return true
			},
			// C
			func(d Dependency) bool {
				t.Log("visited", d)
				assert.Empty(d.Name)
				assert.Equal("buffers", d.Group)
				assert.False(d.Optional)
				assert.Equal(
					reflect.StructTag(`group:"buffers"`),
					d.Tag,
				)

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

func TestValueOf(t *testing.T) {
	testData := []struct {
		v        interface{}
		expected reflect.Value
	}{
		{
			v:        123,
			expected: reflect.ValueOf(123),
		},
		{
			v:        reflect.ValueOf("test"),
			expected: reflect.ValueOf("test"),
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(
				record.expected.Interface(),
				ValueOf(record.v).Interface(),
			)
		})
	}
}

func TestTypeOf(t *testing.T) {
	testData := []struct {
		v        interface{}
		expected reflect.Type
	}{
		{
			v:        "test",
			expected: reflect.TypeOf("test"),
		},
		{
			v:        reflect.ValueOf(123),
			expected: reflect.TypeOf(123),
		},
		{
			v:        reflect.TypeOf(123),
			expected: reflect.TypeOf(123),
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(record.expected, TypeOf(record.v))
		})
	}
}

func testTryConvertFailure(t *testing.T) {
	assert := assert.New(t)

	assert.False(TryConvert(
		"testy mc test",
		func(int) {
			assert.Fail("that is not convertible to an int")
		},
		func(io.Reader) {
			assert.Fail("that is not convertible to an io.Reader")
		},
	))
}

func testTryConvertFunction(t *testing.T) {
	type f1 func(http.Handler) http.Handler
	type f2 func(http.Handler) http.Handler

	t.Run("ScalarToScalar", func(t *testing.T) {
		var (
			assert = assert.New(t)

			expectedCalled    = false
			expected       f1 = func(http.Handler) http.Handler {
				expectedCalled = true
				return nil
			}
		)

		assert.True(TryConvert(
			expected,
			func(int) {
				assert.Fail("that is not convertible to an int")
			},
			func(f f2) {
				f(nil)
			},
		))

		assert.True(expectedCalled)
	})

	t.Run("VectorToVector", func(t *testing.T) {
		var (
			assert = assert.New(t)

			expectedCalled = []bool{false, false, false}
			expected       = []f1{
				func(http.Handler) http.Handler {
					expectedCalled[0] = true
					return nil
				},
				func(http.Handler) http.Handler {
					expectedCalled[1] = true
					return nil
				},
				func(http.Handler) http.Handler {
					expectedCalled[2] = true
					return nil
				},
			}
		)

		assert.True(TryConvert(
			expected,
			func(int) {
				assert.Fail("that is not convertible to an int")
			},
			func(f []f2) {
				for _, e := range f {
					e(nil)
				}
			},
		))

		assert.Equal(
			[]bool{true, true, true},
			expectedCalled,
		)
	})
}

func testTryConvertInterface(t *testing.T) {
	t.Run("ScalarToScalar", func(t *testing.T) {
		var (
			assert = assert.New(t)
			buffer = new(bytes.Buffer)
			actual io.Writer
		)

		assert.True(TryConvert(
			buffer,
			func(v int64) {
				assert.Fail("that is not convertible to an int")
			},
			func(w io.Writer) {
				actual = w
			},
		))

		assert.Equal(buffer, actual)
	})

	t.Run("VectorToVector", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			buffers = []*bytes.Buffer{
				new(bytes.Buffer),
				new(bytes.Buffer),
			}

			converted bool
		)

		assert.True(TryConvert(
			buffers,
			func(v int64) {
				assert.Fail("that is not convertible to an int")
			},
			func(w []io.Writer) {
				converted = true
			},
		))

		assert.True(converted)
	})
}

func testTryConvertValue(t *testing.T) {
	t.Run("ScalarToScalar", func(t *testing.T) {
		var (
			assert = assert.New(t)
			actual int64
		)

		assert.True(TryConvert(
			int(123),
			func(*bytes.Buffer) {
				assert.Fail("that is not convertible to a *bytes.Buffer")
			},
			func(v int64) {
				actual = v
			},
		))

		assert.Equal(int64(123), actual)
	})

	t.Run("VectorToVector", func(t *testing.T) {
		var (
			assert = assert.New(t)
			actual []int64
		)

		assert.True(TryConvert(
			[]int{6, 7, 8},
			func(*bytes.Buffer) {
				assert.Fail("that is not convertible to a *bytes.Buffer")
			},
			func(v []int64) {
				actual = v
			},
		))

		assert.Equal([]int64{6, 7, 8}, actual)
	})
}

func TestTryConvert(t *testing.T) {
	t.Run("Failure", testTryConvertFailure)
	t.Run("Function", testTryConvertFunction)
	t.Run("Interface", testTryConvertInterface)
	t.Run("Value", testTryConvertValue)
}
