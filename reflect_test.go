package arrange

import (
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

func TestIsDependency(t *testing.T) {
	type Dependencies struct {
		fx.In
		unexported   int
		Valid        string
		ZeroValue    string `optional:"true"`
		NotZeroValue string `optional:"true"`
	}

	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = reflect.ValueOf(Dependencies{
			NotZeroValue: "this should be seen as a dependency",
		})
	)

	require.NotPanics(func() {
		// make sure this is a struct
		v.NumField()
	})

	assert.False(IsDependency(v.Type().Field(0), v.Field(0)))
	assert.False(IsDependency(v.Type().Field(1), v.Field(1)))
	assert.True(IsDependency(v.Type().Field(2), v.Field(2)))
	assert.False(IsDependency(v.Type().Field(3), v.Field(3)))
	assert.True(IsDependency(v.Type().Field(4), v.Field(4)))
}

func testVisitFieldsNotAStruct(t *testing.T) {
	testData := []interface{}{
		123,
		new(int),
		new((*int)),
	}

	for i, v := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert = assert.New(t)
				root   = VisitFields(v, func(reflect.StructField, reflect.Value) VisitResult {
					assert.Fail("The visitor should not have been called")
					return VisitTerminate
				})
			)

			assert.False(root.IsValid())
		})
	}
}

func testVisitFieldsSimple(t *testing.T) {
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
		for i, v := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				var (
					assert       = assert.New(t)
					require      = require.New(t)
					actualNames  []string
					actualValues []interface{}
					root         = VisitFields(
						v,
						func(f reflect.StructField, fv reflect.Value) VisitResult {
							actualNames = append(actualNames, f.Name)
							actualValues = append(actualValues, fv.Interface())
							assert.Empty(f.PkgPath)
							assert.Equal(f.Type, fv.Type())
							return VisitContinue
						})
				)

				assert.ElementsMatch(
					[]string{"Value1", "Value2", "Value3", "Value4"},
					actualNames,
				)

				assert.ElementsMatch(
					[]interface{}{123, "test", 3.14, []string{"more", "testing"}},
					actualValues,
				)

				require.True(root.IsValid())
				assert.Equal(reflect.TypeOf(Simple{}), root.Type())
			})
		}
	})

	t.Run("Terminate", func(t *testing.T) {
		for i, simple := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				var (
					assert       = assert.New(t)
					require      = require.New(t)
					actualNames  []string
					actualValues []interface{}
					root         = VisitFields(
						simple,
						func(f reflect.StructField, fv reflect.Value) VisitResult {
							actualNames = append(actualNames, f.Name)
							actualValues = append(actualValues, fv.Interface())
							assert.Empty(f.PkgPath)
							assert.Equal(f.Type, fv.Type())
							return VisitTerminate
						})
				)

				assert.ElementsMatch(
					[]string{"Value1"},
					actualNames,
				)

				assert.ElementsMatch(
					[]interface{}{123},
					actualValues,
				)

				require.True(root.IsValid())
				assert.Equal(reflect.TypeOf(Simple{}), root.Type())
			})
		}
	})
}

func testVisitFieldsEmbedded(t *testing.T) {
	type Leaf struct {
		unexported int
		Leaf1      int
		Leaf2      string
	}

	type Composite struct {
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
					require      = require.New(t)
					actualNames  []string
					actualValues []interface{}
					root         = VisitFields(
						v,
						func(f reflect.StructField, fv reflect.Value) VisitResult {
							actualNames = append(actualNames, f.Name)
							actualValues = append(actualValues, fv.Interface())
							assert.Empty(f.PkgPath)
							assert.Equal(f.Type, fv.Type())
							return VisitContinue
						})
				)

				t.Log("actualNames", actualNames)

				assert.ElementsMatch(
					[]string{"Leaf", "Leaf1", "Leaf2", "Composite1", "Composite2", "Composite3"},
					actualNames,
				)

				assert.ElementsMatch(
					[]interface{}{Leaf{Leaf1: 734, Leaf2: "leafy test"}, 734, "leafy test", 823, "compositey test", Leaf{Leaf1: 111}},
					actualValues,
				)

				require.True(root.IsValid())
				assert.Equal(reflect.TypeOf(Composite{}), root.Type())
			})
		}
	})
}

func TestVisitFields(t *testing.T) {
	t.Run("NotAStruct", testVisitFieldsNotAStruct)
	t.Run("Simple", testVisitFieldsSimple)
	t.Run("Embedded", testVisitFieldsEmbedded)
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

func testTryConvertScalar(t *testing.T) {
	assert := assert.New(t)
	result, ok := TryConvert(int64(0), int(123))
	assert.Equal([]int64{123}, result)
	assert.True(ok)
}

func testTryConvertFunction(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		srcCalled = false
		src       = func(http.ResponseWriter, *http.Request) {
			srcCalled = true
		}
	)

	result, success := TryConvert(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		src,
	)

	require.True(success)

	dst, ok := result.([]http.HandlerFunc)
	require.True(ok)
	require.NotEmpty(dst)
	dst[0](nil, nil)
	assert.True(srcCalled)
}

func testTryConvertArray(t *testing.T) {
	assert := assert.New(t)
	result, ok := TryConvert(int64(0), [4]int{67, -45, 13, 903})
	assert.Equal([]int64{67, -45, 13, 903}, result)
	assert.True(ok)
}

func testTryConvertSlice(t *testing.T) {
	assert := assert.New(t)
	result, ok := TryConvert(int64(0), []int{67, -45, 13, 903})
	assert.Equal([]int64{67, -45, 13, 903}, result)
	assert.True(ok)
}

func testTryConvertFailure(t *testing.T) {
	assert := assert.New(t)
	result, ok := TryConvert((*http.Request)(nil), 45)
	assert.Nil(result)
	assert.False(ok)
}

func TestTryConvert(t *testing.T) {
	t.Run("Scalar", testTryConvertScalar)
	t.Run("Function", testTryConvertFunction)
	t.Run("Array", testTryConvertArray)
	t.Run("Slice", testTryConvertSlice)
	t.Run("Failure", testTryConvertFailure)
}
