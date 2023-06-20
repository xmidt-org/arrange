package arrange

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SafeSuite struct {
	suite.Suite
}

func (suite *SafeSuite) TestNilPointer() {
	var (
		candidate *int
		def       = 123
	)

	suite.Equal(
		&def,
		Safe(candidate, &def),
	)
}

func TestSafe(t *testing.T) {
	suite.Run(t, new(SafeSuite))
}
