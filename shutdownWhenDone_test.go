package arrange

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetest"
	"go.uber.org/fx"
)

type ShutdownWhenDoneSuite struct {
	suite.Suite
}

func (suite *ShutdownWhenDoneSuite) testShutdownWhenDoneNoError() {
	var (
		control = make(chan struct{})
		app     = arrangetest.NewApp(
			suite,
			fx.Invoke(
				func(sh fx.Shutdowner) {
					go ShutdownWhenDone(
						sh,
						func(err error) int {
							suite.NoError(err)
							return 123
						},
						func() {
							<-control
						},
					)
				},
			),
		)
	)

	app.RequireStart()
	close(control)
	select {
	case sig := <-app.Wait():
		suite.Equal(123, sig.ExitCode)
	case <-time.After(time.Second):
		suite.Fail("did not receive shutdown signal")
	}
}

func (suite *ShutdownWhenDoneSuite) testShutdownWhenDoneWithError() {
	var (
		expectedErr = errors.New("expected")
		control     = make(chan struct{})
		app         = arrangetest.NewApp(
			suite,
			fx.Invoke(
				func(sh fx.Shutdowner) {
					go ShutdownWhenDone(
						sh,
						func(err error) int {
							suite.Same(expectedErr, err)
							return 123
						},
						func() error {
							<-control
							return expectedErr
						},
					)
				},
			),
		)
	)

	app.RequireStart()
	close(control)
	select {
	case sig := <-app.Wait():
		suite.Equal(123, sig.ExitCode)
	case <-time.After(time.Second):
		suite.Fail("did not receive shutdown signal")
	}
}

func (suite *ShutdownWhenDoneSuite) TestShutdownWhenDone() {
	suite.Run("NoError", suite.testShutdownWhenDoneNoError)
	suite.Run("WithError", suite.testShutdownWhenDoneWithError)
}

func (suite *ShutdownWhenDoneSuite) testShutdownWhenDoneCtxNoError() {
	var (
		expectedCtx = context.WithValue(context.Background(), "foo", "bar")
		control     = make(chan struct{})
		app         = arrangetest.NewApp(
			suite,
			fx.Invoke(
				func(sh fx.Shutdowner) {
					go ShutdownWhenDoneCtx(
						expectedCtx,
						sh,
						func(err error) int {
							suite.NoError(err)
							return 123
						},
						func(ctx context.Context) {
							suite.Same(expectedCtx, ctx)
							<-control
						},
					)
				},
			),
		)
	)

	app.RequireStart()
	close(control)
	select {
	case sig := <-app.Wait():
		suite.Equal(123, sig.ExitCode)
	case <-time.After(time.Second):
		suite.Fail("did not receive shutdown signal")
	}
}

func (suite *ShutdownWhenDoneSuite) testShutdownWhenDoneCtxWithError() {
	var (
		expectedCtx = context.WithValue(context.Background(), "foo", "bar")
		expectedErr = errors.New("expected")
		control     = make(chan struct{})
		app         = arrangetest.NewApp(
			suite,
			fx.Invoke(
				func(sh fx.Shutdowner) {
					go ShutdownWhenDoneCtx(
						expectedCtx,
						sh,
						func(err error) int {
							suite.Same(expectedErr, err)
							return 123
						},
						func(ctx context.Context) error {
							suite.Same(expectedCtx, ctx)
							<-control
							return expectedErr
						},
					)
				},
			),
		)
	)

	app.RequireStart()
	close(control)
	select {
	case sig := <-app.Wait():
		suite.Equal(123, sig.ExitCode)
	case <-time.After(time.Second):
		suite.Fail("did not receive shutdown signal")
	}
}

func (suite *ShutdownWhenDoneSuite) TestShutdownWhenDoneCtx() {
	suite.Run("NoError", suite.testShutdownWhenDoneCtxNoError)
	suite.Run("WithError", suite.testShutdownWhenDoneCtxWithError)
}

func TestShutdownWhenDone(t *testing.T) {
	suite.Run(t, new(ShutdownWhenDoneSuite))
}
