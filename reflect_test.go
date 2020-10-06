package arrange

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestIsIn(t *testing.T) {
	type SimpleIn struct {
		fx.In

		Foo int
		bar string
	}

	type NestedIn struct {
		SimpleIn
		Another float64
	}

	type FieldIn struct {
		Test fx.In
	}

	type NotIn struct {
		Name string
		Age  int
	}

	testData := []struct {
		input     interface{}
		inspected reflect.Type
		expected  bool
	}{
		{
			input:     SimpleIn{},
			inspected: reflect.TypeOf(SimpleIn{}),
			expected:  true,
		},
		{
			input:     reflect.TypeOf(SimpleIn{}),
			inspected: reflect.TypeOf(SimpleIn{}),
			expected:  true,
		},
		{
			input:     reflect.ValueOf(SimpleIn{}),
			inspected: reflect.TypeOf(SimpleIn{}),
			expected:  true,
		},
		{
			input:     NestedIn{},
			inspected: reflect.TypeOf(NestedIn{}),
			expected:  true,
		},
		{
			input:     reflect.TypeOf(NestedIn{}),
			inspected: reflect.TypeOf(NestedIn{}),
			expected:  true,
		},
		{
			input:     reflect.ValueOf(NestedIn{}),
			inspected: reflect.TypeOf(NestedIn{}),
			expected:  true,
		},
		{
			input:     FieldIn{},
			inspected: reflect.TypeOf(FieldIn{}),
			expected:  false,
		},
		{
			input:     NotIn{},
			inspected: reflect.TypeOf(NotIn{}),
			expected:  false,
		},
		{
			input:     new(int),
			inspected: reflect.TypeOf((*int)(nil)),
			expected:  false,
		},
		{
			input:     "this is most certainly not an fx.In struct",
			inspected: reflect.TypeOf(""),
			expected:  false,
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert         = assert.New(t)
				actual, result = IsIn(record.input)
			)

			assert.Equal(record.inspected, actual)
			assert.Equal(record.expected, result)
		})
	}
}

func TestIsInjected(t *testing.T) {
	type Dependencies struct {
		// NOTE: IsOptional doesn't depend on embedding fx.In

		Simple   *bytes.Buffer
		Required *bytes.Buffer `optional:"false"`
		Optional *bytes.Buffer `optional:"true"`
	}

	var (
		assert = assert.New(t)
		actual = map[string]bool{}
	)

	VisitDependencies(
		Dependencies{},
		func(f reflect.StructField, fv reflect.Value) bool {
			actual[f.Name] = IsInjected(f, fv)
			return true
		},
	)

	assert.Equal(
		map[string]bool{
			"Simple":   true,
			"Required": true,
			"Optional": false,
		},
		actual,
	)
}

func testVisitDependenciesNotAStruct(t *testing.T) {
	testData := []interface{}{
		123,
		new(int),
		new((*int)),
	}

	for i, root := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert := assert.New(t)
			VisitDependencies(root, func(reflect.StructField, reflect.Value) bool {
				assert.Fail("The visitor should not have been called")
				return false
			})
		})
	}
}

func testVisitDependenciesSimple(t *testing.T) {
	type Simple struct {
		unexported string
		Value1     int
		Value2     string
		Value3     float64
		Value4     []string
	}

	testData := []interface{}{
		Simple{
			Value1: 123,
			Value2: "test",
			Value3: 3.14,
			Value4: []string{"more", "testing"},
		},
		reflect.ValueOf(Simple{
			Value1: 123,
			Value2: "test",
			Value3: 3.14,
			Value4: []string{"more", "testing"},
		}),
		&Simple{
			Value1: 123,
			Value2: "test",
			Value3: 3.14,
			Value4: []string{"more", "testing"},
		},
		reflect.ValueOf(&Simple{
			Value1: 123,
			Value2: "test",
			Value3: 3.14,
			Value4: []string{"more", "testing"},
		}),
	}

	t.Run("All", func(t *testing.T) {
		for i, root := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				var (
					assert       = assert.New(t)
					actualNames  []string
					actualValues []interface{}
				)

				VisitDependencies(
					root,
					func(f reflect.StructField, fv reflect.Value) bool {
						actualNames = append(actualNames, f.Name)
						actualValues = append(actualValues, fv.Interface())
						assert.Empty(f.PkgPath)
						assert.Equal(f.Type, fv.Type())
						assert.True(fv.IsValid())
						assert.True(fv.CanInterface())
						return true
					})

				assert.ElementsMatch(
					[]string{"Value1", "Value2", "Value3", "Value4"},
					actualNames,
				)

				assert.ElementsMatch(
					[]interface{}{123, "test", 3.14, []string{"more", "testing"}},
					actualValues,
				)
			})
		}
	})

	t.Run("Terminate", func(t *testing.T) {
		for i, simple := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				var (
					assert       = assert.New(t)
					actualNames  []string
					actualValues []interface{}
				)

				VisitDependencies(
					simple,
					func(f reflect.StructField, fv reflect.Value) bool {
						actualNames = append(actualNames, f.Name)
						actualValues = append(actualValues, fv.Interface())
						assert.Empty(f.PkgPath)
						assert.Equal(f.Type, fv.Type())
						assert.True(fv.IsValid())
						assert.True(fv.CanInterface())
						return false
					})

				assert.ElementsMatch(
					[]string{"Value1"},
					actualNames,
				)

				assert.ElementsMatch(
					[]interface{}{123},
					actualValues,
				)
			})
		}
	})
}

func testVisitDependenciesEmbedded(t *testing.T) {
	type Leaf struct {
		unexported int
		Leaf1      int
		Leaf2      string
	}

	type Composite struct {
		fx.In      // should never be visited
		Leaf       // embedded, which should be visited and possibly traversed
		Composite1 int
		Composite2 string
		Composite3 Leaf // not embedded and never traversed
	}

	testData := []interface{}{
		Composite{
			Leaf: Leaf{
				Leaf1: 734,
				Leaf2: "leafy test",
			},
			Composite1: 823,
			Composite2: "compositey test",
			Composite3: Leaf{
				Leaf1: 111,
			},
		},
		reflect.ValueOf(Composite{
			Leaf: Leaf{
				Leaf1: 734,
				Leaf2: "leafy test",
			},
			Composite1: 823,
			Composite2: "compositey test",
			Composite3: Leaf{
				Leaf1: 111,
			},
		}),
		&Composite{
			Leaf: Leaf{
				Leaf1: 734,
				Leaf2: "leafy test",
			},
			Composite1: 823,
			Composite2: "compositey test",
			Composite3: Leaf{
				Leaf1: 111,
			},
		},
		reflect.ValueOf(&Composite{
			Leaf: Leaf{
				Leaf1: 734,
				Leaf2: "leafy test",
			},
			Composite1: 823,
			Composite2: "compositey test",
			Composite3: Leaf{
				Leaf1: 111,
			},
		}),
	}

	t.Run("All", func(t *testing.T) {
		for i, v := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				var (
					assert       = assert.New(t)
					actualNames  []string
					actualValues []interface{}
				)

				VisitDependencies(
					v,
					func(f reflect.StructField, fv reflect.Value) bool {
						assert.NotEqual(InType(), f.Type)
						actualNames = append(actualNames, f.Name)
						actualValues = append(actualValues, fv.Interface())
						assert.Empty(f.PkgPath)
						assert.Equal(f.Type, fv.Type())
						assert.True(fv.IsValid())
						assert.True(fv.CanInterface())
						return true
					})

				t.Log("actualNames", actualNames)

				assert.ElementsMatch(
					[]string{"Leaf", "Leaf1", "Leaf2", "Composite1", "Composite2", "Composite3"},
					actualNames,
				)

				assert.ElementsMatch(
					[]interface{}{Leaf{Leaf1: 734, Leaf2: "leafy test"}, 734, "leafy test", 823, "compositey test", Leaf{Leaf1: 111}},
					actualValues,
				)
			})
		}
	})
}

func TestVisitDependencies(t *testing.T) {
	t.Run("NotAStruct", testVisitDependenciesNotAStruct)
	t.Run("Simple", testVisitDependenciesSimple)
	t.Run("Embedded", testVisitDependenciesEmbedded)
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

type Attributes interface {
	Get(string) (interface{}, bool)
}

type BasicAttributes map[string]interface{}

func (ba BasicAttributes) Get(key string) (interface{}, bool) {
	v, ok := ba[key]
	return v, ok
}

func TestBascule(t *testing.T) {
	var (
		assert                 = assert.New(t)
		attributes interface{} = BasicAttributes{}
		result     interface{}
	)

	assert.True(TryConvert(
		attributes,
		func(value Attributes) {
			result = value
		},
	))

	assert.Equal(attributes, result)
}
