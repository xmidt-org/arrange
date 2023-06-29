package arrangetest

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TLSSuiteSuite is a sweet, sweet testing suite that simply ensure
// the TLSSuite's lifecycle works properly.
type TLSSuiteSuite struct {
	TLSSuite
}

func (suite *TLSSuiteSuite) TestState() {
	suite.NotNil(suite.certificate)
	suite.FileExists(suite.certificateFile)
	suite.FileExists(suite.keyFile)
}

func TestTLSSuite(t *testing.T) {
	suite.Run(t, new(TLSSuiteSuite))
}
