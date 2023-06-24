package arrangehttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ServerSuite struct {
	suite.Suite
}

func (suite *ServerSuite) TestNewServer() {
	var server *http.Server
	app := fxtest.New(
		suite.T(),
		fx.Supply(ServerConfig{
			Address: ":1234",
		}),
		fx.Provide(
			fx.Annotate(
				NewServer,
				fx.ParamTags("", `optional:"true"`),
			),
		),
		fx.Populate(&server),
	)

	app.RequireStart()
	app.RequireStop()

	suite.Require().NotNil(server)
	suite.Equal(":1234", server.Addr)
}

func (suite *ServerSuite) TestProvideServer() {
	app := fxtest.New(
		suite.T(),
		ProvideServer("main"),
	)

	app.RequireStart()
	app.RequireStop()
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}
