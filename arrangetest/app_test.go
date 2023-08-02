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

package arrangetest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
)

type AppSuite struct {
	suite.Suite
}

func (suite *AppSuite) testNewAppSuite() {
	var value int
	NewApp(
		suite,
		fx.Provide(
			func() int { return 123 },
		),
		fx.Populate(&value),
	)

	suite.Equal(123, value)
}

func (suite *AppSuite) testNewAppTest() {
	var value int
	NewApp(
		suite.T(),
		fx.Provide(
			func() int { return 123 },
		),
		fx.Populate(&value),
	)

	suite.Equal(123, value)
}

func (suite *AppSuite) TestNewApp() {
	suite.Run("Suite", suite.testNewAppSuite)
	suite.Run("Test", suite.testNewAppTest)
}

func (suite *AppSuite) testNewErrAppSuccess() {
	var value int
	NewErrApp(
		suite,
		fx.Provide(
			func() (int, error) {
				return 0, errors.New("this should be successful, as an error did happen when creating the App")
			},
		),
		fx.Populate(&value), // force the constructor to run
	)
}

func (suite *AppSuite) testNewErrAppFail() {
	mockT := new(mockTestable)
	mockT.ExpectAnyErrorf()

	NewErrApp(mockT) // no error should cause an assert failure, which is a success for this test

	mockT.AssertExpectations(suite.T())
}

func (suite *AppSuite) TestNewErrApp() {
	suite.Run("Success", suite.testNewErrAppSuccess)
	suite.Run("Fail", suite.testNewErrAppFail)
}

func TestApp(t *testing.T) {
	suite.Run(t, new(AppSuite))
}
