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

package arrangetls

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// SuiteSuite is a sweet, sweet testing suite that simply ensure
// the TLSSuite's lifecycle works properly.
type SuiteSuite struct {
	Suite
}

func (suite *SuiteSuite) TestState() {
	suite.NotNil(suite.certificate)
	suite.FileExists(suite.certificateFile)
	suite.FileExists(suite.keyFile)
}

func (suite *SuiteSuite) TestConfig() {
	suite.NotNil(
		suite.Config(),
	)
}

func (suite *SuiteSuite) TestTLSConfig() {
	suite.NotNil(
		suite.TLSConfig(),
	)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteSuite))
}
