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
	"testing"

	"github.com/stretchr/testify/suite"
)

type TestableSuite struct {
	suite.Suite
}

func (suite *TestableSuite) testAsTestableInvalidValue() {
	suite.Panics(func() {
		AsTestable(123)
	})
}

func (suite *TestableSuite) testAsTestableWithSuite() {
	suite.NotNil(
		AsTestable(suite),
	)
}

func (suite *TestableSuite) testAsTestableWithTestingT() {
	suite.NotNil(
		AsTestable(suite.T()),
	)
}

func (suite *TestableSuite) TestAsTestable() {
	suite.Run("InvalidValue", suite.testAsTestableInvalidValue)
	suite.Run("WithSuite", suite.testAsTestableWithSuite)
	suite.Run("WithTestingT", suite.testAsTestableWithTestingT)
}

func TestTestable(t *testing.T) {
	suite.Run(t, new(TestableSuite))
}
