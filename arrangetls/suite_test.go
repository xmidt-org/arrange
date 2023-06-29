package arrangetls

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// SuiteSuite is a sweet, sweet testing suite that simply ensure
// the TLSSuite's lifecycle works properly.
type SuiteSuite struct {
	Suite
}

func (suite *SuiteSuite) TestState() {
	suite.NotNil(suite.certificate)
	suite.FileExists(suite.certificateFile)
	suite.FileExists(suite.keyFile)
}

func (suite *SuiteSuite) TestTLSConfig() {
	suite.NotNil(
		suite.TLSConfig(),
	)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteSuite))
}
