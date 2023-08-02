// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangereflect

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SafeSuite struct {
	suite.Suite
}

func (suite *SafeSuite) TestSimple() {
	var (
		candidate = 123
		def       = 456
	)

	suite.Equal(
		123,
		Safe(candidate, def),
	)
}

func (suite *SafeSuite) TestNilPointer() {
	var (
		candidate *int
		def       = 123
	)

	suite.Equal(
		&def,
		Safe(candidate, &def),
	)
}

func (suite *SafeSuite) TestNonNilPointer() {
	var (
		candidate = 123
		def       = 456
	)

	suite.Equal(
		&candidate,
		Safe(&candidate, &def),
	)
}

func (suite *SafeSuite) TestUninitializedInterface() {
	var (
		candidate http.HandlerFunc
		actual    = Safe[http.Handler](candidate, http.DefaultServeMux)
	)

	suite.IsType(
		http.DefaultServeMux,
		actual,
	)
}

func TestSafe(t *testing.T) {
	suite.Run(t, new(SafeSuite))
}
