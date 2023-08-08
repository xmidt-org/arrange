package arrangehttp

import (
	"errors"
	"github.com/xmidt-org/arrange/arrangeoption"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/arrange/arrangetest"
	"github.com/xmidt-org/httpaux"
	"go.uber.org/fx"
)

type ServerSuite struct {
	suite.Suite
}

// newConstantHandler creates an httpaux.ConstantHandler with a known, non-standard status code.
func (suite *ServerSuite) newConstantHandler() httpaux.ConstantHandler {
	return httpaux.ConstantHandler{
		StatusCode: 299,
	}
}

// supplyConstantHandler uses fx.Supply to emit the handler returned by newConstantHandler
// into the enclosing fx.App.
func (suite *ServerSuite) supplyConstantHandler(anns ...fx.Annotation) fx.Option {
	return fx.Supply(
		fx.Annotate(
			suite.newConstantHandler(),
			anns...,
		),
	)
}

// runHandler runs the ServeHTTP method of the given handler, returning the response.  If
// request is nil, it is assumed the request doesn't matter and a generic request is used instead.
func (suite *ServerSuite) runHandler(h http.Handler, request *http.Request) *httptest.ResponseRecorder {
	suite.Require().NotNil(h)
	if request == nil {
		request = httptest.NewRequest("GET", "/", nil)
	}

	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	return response
}

// assertUsesConstantHandler executes a request against the given server's Handler and asserts
// that it came from the handler returned by newConstantHandler.  If the given request is nil,
// a default one is used instead.  The test response is returned for any desired further
// assertions.
func (suite *ServerSuite) assertUsesConstantHandler(server *http.Server, request *http.Request) *httptest.ResponseRecorder {
	suite.Require().NotNil(server)
	return suite.runHandler(server.Handler, request)
}

func (suite *ServerSuite) testNewServerNoOptions() {
	var server *http.Server
	app := arrangetest.NewApp(
		suite,
		fx.Supply(ServerConfig{}),
		suite.supplyConstantHandler(fx.As(new(http.Handler))),
		fx.Provide(
			NewServer,
		),
		fx.Populate(&server),
	)

	app.RequireStart()
	app.RequireStop()
	suite.assertUsesConstantHandler(server, nil)
}

func (suite *ServerSuite) testNewServerWithOptions() {
	var server *http.Server
	app := arrangetest.NewApp(
		suite,
		fx.Supply(ServerConfig{
			Header: http.Header{
				"Custom": []string{"true"},
			},
		}),
		suite.supplyConstantHandler(fx.As(new(http.Handler))),
		fx.Provide(
			fx.Annotate(
				func() arrangeoption.Option[http.Server] {
					return arrangeoption.AsOption[http.Server](func(s *http.Server) {
						s.ReadTimeout = 27 * time.Second
					})
				},
				arrange.Tags().Group("options").ResultTags(),
			),
			fx.Annotate(
				func() arrangeoption.Option[http.Server] {
					return arrangeoption.AsOption[http.Server](func(s *http.Server) {
						s.WriteTimeout = 345 * time.Minute
					})
				},
				arrange.Tags().Group("options").ResultTags(),
			),
			fx.Annotate(
				NewServer,
				arrange.Tags().
					Skip().
					Skip().
					Group("options").
					ParamTags(),
			),
		),
		fx.Populate(&server),
	)

	app.RequireStart()
	app.RequireStop()
	response := suite.assertUsesConstantHandler(server, nil)
	suite.Equal("true", response.Result().Header.Get("Custom"))
	suite.Equal(27*time.Second, server.ReadTimeout)
	suite.Equal(345*time.Minute, server.WriteTimeout)
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
	var server *http.Server
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

func (suite *ServerSuite) testProvideServerFull() {
	var server *http.Server
	capture := make(chan net.Addr, 1)
	app := arrangetest.NewApp(
		suite,
		suite.supplyConstantHandler(fx.As(new(http.Handler))),
		fx.Supply(
			fx.Annotated{
				Target: ServerConfig{
					ReadTimeout: 27 * time.Second,
				},
				Name: "test.config",
			},
			fx.Annotated{
				Target: arrangetest.ListenCapture(capture),
				Name:   "test.listener.middleware",
			},
		),
		ProvideServer(
			"test",
			// verify that external options work:
			arrangeoption.AsOption[http.Server](func(s *http.Server) {
				s.WriteTimeout = 23973 * time.Hour
			}),
		),
		fx.Populate(
			fx.Annotate(
				&server,
				arrange.Tags().Name("test").ParamTags(),
			),
		),
	)

	app.RequireStart()
	suite.assertUsesConstantHandler(server, nil)
	suite.Equal(27*time.Second, server.ReadTimeout)
	suite.Equal(23973*time.Hour, server.WriteTimeout)

	app.RequireStop()
}

func (suite *ServerSuite) testProvideServerInvalidExternalValue() {
	var server *http.Server
	arrangetest.NewErrApp(
		suite,
		ProvideServer("test", "this is not a valid external value"),
		fx.Populate(&server), // needed to force the constructor to run
	)
}

func (suite *ServerSuite) testProvideServerAbnormalServerExit() {
	var (
		server       *http.Server
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
	suite.Run("Full", suite.testProvideServerFull)
	suite.Run("InvalidExternalValue", suite.testProvideServerInvalidExternalValue)
	suite.Run("AbnormalServerExit", suite.testProvideServerAbnormalServerExit)
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}
