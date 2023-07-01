package arrangehttp

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ClientSuite struct {
	suite.Suite
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}
