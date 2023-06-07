package arrangehttp

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServerSuite struct {
	suite.Suite
}

func (suite *ServerSuite) TestApplyServerOptions() {
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}
