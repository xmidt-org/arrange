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

func (suite *ServerSuite) TestApplyServerOptions() {
	var (
		expectedServer = new(http.Server)
		actualServer   *http.Server

		mock0 = new(mockOption[http.Server])
		mock1 = new(mockOption[http.Server])
	)

	mock0.ExpectApply(expectedServer).Return(nil)
	mock1.ExpectApply(expectedServer).Return(nil)

	app := fxtest.New(
		suite.T(),
		fx.Provide(
			func() *http.Server {
				return expectedServer
			},
			fx.Annotate(
				func() Option[http.Server] { return mock0 },
				fx.ResultTags(`group:"options"`),
			),
			fx.Annotate(
				func() Option[http.Server] { return mock1 },
				fx.ResultTags(`group:"options"`),
			),
		),
		fx.Decorate(
			fx.Annotate(
				ApplyServerOptions,
				fx.ParamTags("", `group:"options"`),
			),
		),
		fx.Populate(&actualServer),
	)

	app.RequireStart()
	app.RequireStop()

	suite.Same(expectedServer, actualServer)
	mock0.AssertExpectations(suite.T())
	mock1.AssertExpectations(suite.T())
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
