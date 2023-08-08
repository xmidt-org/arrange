package arrangetest

import "github.com/stretchr/testify/suite"

type OptionSuite[T any] struct {
	suite.Suite
	Target *T
}

func (suite *OptionSuite[T]) SetupTest() {
	suite.Target = new(T)
}

func (suite *OptionSuite[T]) SetupSubTest() {
	suite.Target = new(T)
}
