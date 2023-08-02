/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
	type contextKey struct{}

	var (
		expectedCtx = context.WithValue(context.Background(), contextKey{}, "bar")
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
	type contextKey struct{}

	var (
		expectedCtx = context.WithValue(context.Background(), contextKey{}, "bar")
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
