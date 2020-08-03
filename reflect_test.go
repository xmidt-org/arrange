package arrange

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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
