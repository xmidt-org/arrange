package arrangegrpc

import (
	"errors"
	testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangeoption"
	"github.com/xmidt-org/arrange/arrangetest"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

type ServerSuite struct {
	suite.Suite
}

func (suite *ServerSuite) testNewServerNoOptions() {
	var server *grpc.Server
	app := arrangetest.NewApp(
		suite,
		fx.Supply(ServerConfig{}),
		fx.Provide(
			fx.Annotate(
				NewServer,
				arrange.Tags().
					Skip().
					Group("interceptors").
					Group("server.options").
					Group("options").
					ParamTags(),
			),
		),
		fx.Populate(&server),
	)

	app.RequireStart()
	app.RequireStop()
}

func (suite *ServerSuite) testNewServerWithOptions() {
	var server *grpc.Server
	app := arrangetest.NewApp(
		suite,
		fx.Supply(ServerConfig{}),
		fx.Provide(
			fx.Annotate(
				func() arrangeoption.Option[grpc.Server] {
					return arrangeoption.AsOption[grpc.Server](func(s *grpc.Server) {
						testproto.RegisterTestServiceServer(s, &examplePing{})
					})
				},
				arrange.Tags().Group("options").ResultTags(),
			),
			fx.Annotate(
				NewServer,
				arrange.Tags().
					Skip().
					Group("interceptors").
					Group("server.options").
					Group("options").
					ParamTags(),
			),
		),
		fx.Populate(&server),
	)

	app.RequireStart()
	app.RequireStop()
	_, ok := server.GetServiceInfo()["mwitkow.testproto.TestService"]
	suite.True(ok)
}

func (suite *ServerSuite) TestNewServer() {
	suite.Run("NoOptions", suite.testNewServerNoOptions)
	suite.Run("WithOptions", suite.testNewServerWithOptions)
}

func (suite *ServerSuite) testProvideServerNoName() {
	arrangetest.NewErrApp(
		suite,
		ProvideServer(""), // error
	)
}

func (suite *ServerSuite) testProvideServerSimple() {
	var server *grpc.Server
	capture := make(chan net.Addr, 1)
	app := arrangetest.NewApp(
		suite,
		fx.Supply(
			fx.Annotated{
				Target: ServerConfig{
					Address: ":0",
				},
				Name: "test.config",
			},
		),
		ProvideServer("test", arrangetest.ListenCapture(capture)),
		fx.Populate(
			fx.Annotate(
				&server,
				arrange.Tags().Name("test").ParamTags(),
			),
		),
	)

	app.RequireStart()
	arrangetest.ListenReceive(suite, capture, time.Second)
	app.RequireStop()
	suite.NotNil(server)
}

func (suite *ServerSuite) testProvideServerInvalidExternalValue() {
	var server *grpc.Server
	arrangetest.NewErrApp(
		suite,
		ProvideServer("test", "this is not a valid external value"),
		fx.Populate(&server), // needed to force the constructor to run
	)
}

func (suite *ServerSuite) testProvideServerAbnormalServerExit() {
	var (
		server       *grpc.Server
		expectedErr  = errors.New("expected")
		mockListener = new(arrangetest.MockListener)

		app = arrangetest.NewApp(
			suite,
			ProvideServer(
				"test",
				func(l net.Listener) net.Listener {
					// replace with a misbehaving listener
					l.Close()
					return mockListener
				},
			),
			fx.Populate(
				fx.Annotate(
					&server,
					arrange.Tags().Name("test").ParamTags(),
				),
			),
		)
	)
	l, err := net.Listen("tcp", ":0")
	suite.Require().NoError(err)
	defer l.Close()
	mockListener.ExpectAddr(l.Addr())
	mockListener.ExpectAccept(nil, expectedErr)
	mockListener.ExpectClose(nil)

	app.RequireStart()
	select {
	case signal := <-app.Wait():
		suite.Equal(ServerAbnormalExitCode, signal.ExitCode)
		mockListener.AssertExpectations(suite.T())

	case <-time.After(time.Second):
		suite.Fail("did not receive an fx.ShutdownSignal")
	}
}

func (suite *ServerSuite) TestProvideServer() {
	suite.Run("NoName", suite.testProvideServerNoName)
	suite.Run("Simple", suite.testProvideServerSimple)
	suite.Run("InvalidExternalValue", suite.testProvideServerInvalidExternalValue)
	suite.Run("AbnormalServerExit", suite.testProvideServerAbnormalServerExit)
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}
