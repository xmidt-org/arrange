package arrangetest

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TestableSuite struct {
	suite.Suite
}

func (suite *TestableSuite) testAsTestableInvalidValue() {
	suite.Panics(func() {
		AsTestable(123)
	})
}

func (suite *TestableSuite) testAsTestableWithSuite() {
	suite.NotNil(
		AsTestable(suite),
	)
}

func (suite *TestableSuite) testAsTestableWithTestingT() {
	suite.NotNil(
		AsTestable(suite.T()),
	)
}

func (suite *TestableSuite) TestAsTestable() {
	suite.Run("InvalidValue", suite.testAsTestableInvalidValue)
	suite.Run("WithSuite", suite.testAsTestableWithSuite)
	suite.Run("WithTestingT", suite.testAsTestableWithTestingT)
}

func TestTestable(t *testing.T) {
	suite.Run(t, new(TestableSuite))
}
