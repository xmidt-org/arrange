package arrangegrpc

import (
	"context"
	pb "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetls"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

type simpleServerFactory struct {
	Address   string
	returnErr error
}

func (ssf simpleServerFactory) NewServer(middlewares []grpc.UnaryServerInterceptor, options ...grpc.ServerOption) (*grpc.Server, error) {
	if ssf.returnErr != nil {
		return nil, ssf.returnErr
	}

	return grpc.NewServer(options...), nil
}

type examplePing struct {
	pb.UnimplementedTestServiceServer
}

func (examplePing) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Value: "Pong"}, nil
}

type ServerConfigSuite struct {
	arrangetls.Suite
}

func (suite *ServerConfigSuite) testListenDefault() {
	var (
		// all defaults in the ServerConfig
		l, err = ServerConfig{}.Listen(
			context.Background(),
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()

	suite.IsType((*net.TCPListener)(nil), l)
}

func (suite *ServerConfigSuite) testListenNoTLS() {
	var (
		l, err = ServerConfig{
			Network:   "tcp",
			KeepAlive: 2 * time.Minute,
		}.Listen(
			context.Background(),
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()

	suite.IsType((*net.TCPListener)(nil), l)
}

func (suite *ServerConfigSuite) testListenTLS() {
	var (
		l, err = ServerConfig{
			KeepAlive: time.Minute,
			TLS:       suite.Config(),
		}.Listen(
			context.Background(),
		)
	)

	suite.Require().NoError(err)
	suite.Require().NotNil(l)
	defer l.Close()

	_, isTCP := l.(*net.TCPListener)
	suite.False(isTCP)
}

func (suite *ServerConfigSuite) TestListen() {
	suite.Run("Default", suite.testListenDefault)
	suite.Run("NoTLS", suite.testListenNoTLS)
	suite.Run("TLS", suite.testListenTLS)
}

func TestServerConfig(t *testing.T) {
	suite.Run(t, new(ServerConfigSuite))
}
