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
	"github.com/stretchr/testify/suite"
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
		suite.Equal(InType(), f.Type)
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

type InjectTestSuite struct {
	suite.Suite
}

func (suite *InjectTestSuite) TestEmpty() {
	suite.Empty(Inject{}.Types())

	ft := Inject{}.FuncOf()
	suite.Require().NotNil(ft)
	suite.Require().Equal(reflect.Func, ft.Kind())
	suite.Equal(0, ft.NumIn())
	suite.Equal(0, ft.NumOut())
}

func (suite *InjectTestSuite) TestFuncOf() {
	var (
		s = Struct{}.In().Append(Field{
			Name: "component",
			Type: (*bytes.Buffer)(nil),
		})

		ij = Inject{}.Append(
			(*bytes.Buffer)(nil),
			s.Of(),
		)

		ft = ij.FuncOf(reflect.TypeOf(int(0)), ErrorType())
	)

	suite.Require().NotNil(ft)
	suite.Require().Equal(reflect.Func, ft.Kind())

	suite.Equal(2, ft.NumIn())
	suite.Equal(reflect.TypeOf((*bytes.Buffer)(nil)), ft.In(0))
	suite.Equal(s.Of(), ft.In(1))

	suite.Equal(2, ft.NumOut())
	suite.Equal(reflect.TypeOf(int(0)), ft.Out(0))
	suite.Equal(ErrorType(), ft.Out(1))
}

func (suite *InjectTestSuite) TestMakeFunc() {
	var (
		buffer = new(bytes.Buffer)

		ij = Inject{
			(*bytes.Buffer)(nil),
			reflect.ValueOf(int(0)),
		}

		fv = ij.MakeFunc(
			func(inputs []reflect.Value) (string, error) {
				suite.Require().Len(inputs, 2)
				suite.Equal(buffer, inputs[0].Interface())
				suite.Equal(123, inputs[1].Interface())
				return "test", nil
			},
		)
	)

	outputs := fv.Call(
		[]reflect.Value{
			reflect.ValueOf(buffer),
			reflect.ValueOf(123),
		},
	)

	suite.Require().Len(outputs, 2)
	suite.Equal("test", outputs[0].Interface())
	suite.True(outputs[1].IsNil())
}

func (suite *InjectTestSuite) TestExtend() {
	ij := Inject{}.Extend(
		Inject{
			(*bytes.Buffer)(nil),
			123,
		},
	)

	suite.Require().Len(ij, 2)
	suite.Equal(
		ij,
		Inject{
			(*bytes.Buffer)(nil),
			123,
		},
	)
}

func TestInject(t *testing.T) {
	suite.Run(t, new(InjectTestSuite))
}
