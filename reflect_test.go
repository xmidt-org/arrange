package arrange

import (
	"bytes"
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
				target.Component(),
				record,
			)

			assert.Equal(
				target.ComponentType(),
				reflect.TypeOf(NewTargetTester{}),
			)

			assert.Equal(
				record,
				*target.UnmarshalTo().(*NewTargetTester),
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
				target.Component(),
				&record,
			)

			assert.Equal(
				target.ComponentType(),
				reflect.TypeOf((*NewTargetTester)(nil)),
			)

			assert.Equal(
				&record,
				target.UnmarshalTo().(*NewTargetTester),
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
		target.Component(),
		&NewTargetTester{},
	)

	assert.Equal(
		target.ComponentType(),
		reflect.TypeOf((*NewTargetTester)(nil)),
	)

	assert.Equal(
		&NewTargetTester{},
		target.UnmarshalTo().(*NewTargetTester),
	)
}

func TestNewTarget(t *testing.T) {
	t.Run("Value", testNewTargetValue)
	t.Run("Pointer", testNewTargetPointer)
	t.Run("Nil", testNewTargetNil)
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

func testIsInNotIn(t *testing.T) {
	type Plain struct {
		Value1 string
	}

	testData := []interface{}{
		Plain{
			Value1: "does not matter",
		},
		reflect.ValueOf(Plain{
			Value1: "does not matter",
		}),
		&Plain{
			Value1: "does not matter",
		},
		reflect.ValueOf(&Plain{
			Value1: "does not matter",
		}),
	}

	for i, v := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert   = assert.New(t)
				require  = require.New(t)
				root, ok = IsIn(v)
			)

			assert.False(ok)
			require.True(root.IsValid())
			assert.Equal(reflect.TypeOf(Plain{}), root.Type())
		})
	}
}

func testIsInNotAStruct(t *testing.T) {
	testData := []interface{}{
		123,
		reflect.ValueOf(nil),
		new(int),
		new((*int)),
	}

	for i, v := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert   = assert.New(t)
				root, ok = IsIn(v)
			)

			assert.False(ok)
			assert.False(root.IsValid())
		})
	}
}

func testIsInSuccess(t *testing.T) {
	type Dependencies struct {
		fx.In
		Something    string
		AnotherThing bytes.Buffer
	}

	testData := []interface{}{
		Dependencies{},
		reflect.ValueOf(Dependencies{}),
		&Dependencies{},
		reflect.ValueOf(&Dependencies{}),
	}

	for i, v := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert   = assert.New(t)
				require  = require.New(t)
				root, ok = IsIn(v)
			)

			assert.True(ok)
			require.True(root.IsValid())
			assert.Equal(reflect.TypeOf(Dependencies{}), root.Type())
		})
	}
}

func TestIsIn(t *testing.T) {
	t.Run("NotIn", testIsInNotIn)
	t.Run("NotAStruct", testIsInNotAStruct)
	t.Run("Success", testIsInSuccess)
}
